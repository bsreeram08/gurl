package env

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sreeram/gurl/internal/project"
)

const activeEnvFileName = ".active-env"

type FileEnvStore struct {
	project *project.Project
}

func NewFileEnvStore(proj *project.Project) *FileEnvStore {
	return &FileEnvStore{project: proj}
}

func (s *FileEnvStore) Enabled() bool {
	return s != nil && s.project != nil && s.project.EnvironmentsDir() != ""
}

func (s *FileEnvStore) SaveEnv(env *Environment) error {
	if !s.Enabled() {
		return fmt.Errorf("file environment storage is not configured")
	}
	if env == nil {
		return fmt.Errorf("environment cannot be nil")
	}
	if env.Name == "" {
		return fmt.Errorf("environment name cannot be empty")
	}

	now := time.Now().Unix()
	if env.ID == "" {
		if existing, err := s.GetEnvByName(env.Name); err == nil && existing != nil {
			return fmt.Errorf("environment %q already exists", env.Name)
		}
		env.ID = uuid.New().String()
	}

	existing, oldPath, err := s.findEnvByID(env.ID)
	if err == nil && existing != nil && env.CreatedAt == 0 {
		env.CreatedAt = existing.CreatedAt
	}
	if existingByName, err := s.GetEnvByName(env.Name); err == nil && existingByName.ID != env.ID {
		return fmt.Errorf("environment %q already exists", env.Name)
	}

	if env.CreatedAt == 0 {
		env.CreatedAt = now
	}
	if env.UpdatedAt == 0 {
		env.UpdatedAt = now
	}
	if env.Variables == nil {
		env.Variables = make(map[string]string)
	}
	if env.SecretKeys == nil {
		env.SecretKeys = make(map[string]bool)
	}

	stored, err := envForFileStorage(env)
	if err != nil {
		return err
	}
	targetPath := filepath.Join(s.project.EnvironmentsDir(), safeEnvFileName(env.Name))
	if err := writeEnvJSONFile(targetPath, stored); err != nil {
		return err
	}
	if oldPath != "" && oldPath != targetPath {
		if err := os.Remove(oldPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove old environment file: %w", err)
		}
	}
	return nil
}

func (s *FileEnvStore) GetEnv(id string) (*Environment, error) {
	env, _, err := s.findEnvByID(id)
	if err != nil {
		return nil, fmt.Errorf("environment not found: %s", id)
	}
	return env, nil
}

func (s *FileEnvStore) GetEnvByName(name string) (*Environment, error) {
	envs, err := s.ListEnvs()
	if err != nil {
		return nil, err
	}
	for _, env := range envs {
		if env.Name == name {
			return env, nil
		}
	}
	return nil, fmt.Errorf("environment not found: %s", name)
}

func (s *FileEnvStore) DeleteEnv(id string) error {
	_, path, err := s.findEnvByID(id)
	if err != nil {
		return fmt.Errorf("environment not found: %s", id)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete environment: %w", err)
	}
	return nil
}

func (s *FileEnvStore) ListEnvs() ([]*Environment, error) {
	if !s.Enabled() {
		return []*Environment{}, nil
	}
	entries, err := os.ReadDir(s.project.EnvironmentsDir())
	if err != nil {
		if os.IsNotExist(err) {
			return []*Environment{}, nil
		}
		return nil, fmt.Errorf("failed to read environments directory: %w", err)
	}

	envs := make([]*Environment, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		var env Environment
		if err := readEnvJSONFile(filepath.Join(s.project.EnvironmentsDir(), entry.Name()), &env); err != nil {
			continue
		}
		normalizeEnv(&env)
		if err := decryptFileEnvForUse(&env); err != nil {
			return nil, err
		}
		envs = append(envs, &env)
	}
	sort.SliceStable(envs, func(i, j int) bool {
		return envs[i].Name < envs[j].Name
	})
	return envs, nil
}

func (s *FileEnvStore) GetActiveEnv() (string, error) {
	if !s.Enabled() {
		return "", nil
	}
	data, err := os.ReadFile(filepath.Join(s.project.EnvironmentsDir(), activeEnvFileName))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get active env: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func (s *FileEnvStore) SetActiveEnv(name string) error {
	if !s.Enabled() {
		return fmt.Errorf("file environment storage is not configured")
	}
	path := filepath.Join(s.project.EnvironmentsDir(), activeEnvFileName)
	if name == "" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to clear active env: %w", err)
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create environments directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(name+"\n"), 0644); err != nil {
		return fmt.Errorf("failed to set active env: %w", err)
	}
	return nil
}

func (s *FileEnvStore) HasEnvID(id string) bool {
	if !s.Enabled() || id == "" {
		return false
	}
	_, _, err := s.findEnvByID(id)
	return err == nil
}

func (s *FileEnvStore) HasEnvName(name string) bool {
	if !s.Enabled() || name == "" {
		return false
	}
	_, err := s.GetEnvByName(name)
	return err == nil
}

func (s *FileEnvStore) findEnvByID(id string) (*Environment, string, error) {
	envs, err := s.ListEnvs()
	if err != nil {
		return nil, "", err
	}
	for _, env := range envs {
		if env.ID == id {
			return env, filepath.Join(s.project.EnvironmentsDir(), safeEnvFileName(env.Name)), nil
		}
	}
	return nil, "", fmt.Errorf("environment not found: %s", id)
}

func safeEnvFileName(name string) string {
	return safeEnvPathComponent(name) + ".json"
}

func safeEnvPathComponent(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "environment"
	}
	if isSimpleEnvPathComponent(name) {
		return name
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == '.':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	slug := strings.Trim(builder.String(), "-.")
	if slug == "" {
		slug = "environment"
	}
	sum := sha256.Sum256([]byte(name))
	return slug + "--" + hex.EncodeToString(sum[:])[:8]
}

func isSimpleEnvPathComponent(name string) bool {
	if name == "." || name == ".." || strings.HasPrefix(name, ".") {
		return false
	}
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func normalizeEnv(env *Environment) {
	if env.Variables == nil {
		env.Variables = make(map[string]string)
	}
	if env.SecretKeys == nil {
		env.SecretKeys = make(map[string]bool)
	}
}

func envForFileStorage(source *Environment) (*Environment, error) {
	stored := cloneEnvironment(source)
	if stored == nil || !envHasSecrets(stored) {
		return stored, nil
	}
	key, err := GetOrCreateMachineKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}
	for name, isSecret := range stored.SecretKeys {
		if !isSecret {
			continue
		}
		encrypted, err := EncryptSecret(key, stored.Variables[name])
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt secret %s: %w", name, err)
		}
		stored.Variables[name] = encrypted
	}
	return stored, nil
}

func decryptFileEnvForUse(env *Environment) error {
	if env == nil || !envHasSecrets(env) {
		return nil
	}
	key, err := GetOrCreateMachineKey()
	if err != nil {
		return fmt.Errorf("failed to get encryption key: %w", err)
	}
	for name, isSecret := range env.SecretKeys {
		if isSecret && IsEncryptedValue(env.Variables[name]) {
			decrypted, err := DecryptSecret(key, env.Variables[name])
			if err != nil {
				return fmt.Errorf("failed to decrypt secret %s: %w", name, err)
			}
			env.Variables[name] = decrypted
		}
	}
	return nil
}

func envHasSecrets(env *Environment) bool {
	if env == nil {
		return false
	}
	for _, isSecret := range env.SecretKeys {
		if isSecret {
			return true
		}
	}
	return false
}

func cloneEnvironment(source *Environment) *Environment {
	if source == nil {
		return nil
	}
	clone := *source
	clone.Variables = make(map[string]string, len(source.Variables))
	for key, value := range source.Variables {
		clone.Variables[key] = value
	}
	clone.SecretKeys = make(map[string]bool, len(source.SecretKeys))
	for key, value := range source.SecretKeys {
		clone.SecretKeys[key] = value
	}
	return &clone
}

func readEnvJSONFile(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return nil
}

func writeEnvJSONFile(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	encoder := json.NewEncoder(tmp)
	encoder.SetIndent("", "  ")
	encodeErr := encoder.Encode(value)
	closeErr := tmp.Close()
	if encodeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to encode JSON: %w", encodeErr)
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", closeErr)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to replace JSON file: %w", err)
	}
	return nil
}
