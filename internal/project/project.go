package project

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	DirName       = ".gurl"
	EnvProjectDir = "GURL_PROJECT_DIR"
)

type Project struct {
	Root    string
	GurlDir string
}

func Discover(startDir string, explicitDir string) (*Project, error) {
	if explicitDir == "" {
		explicitDir = os.Getenv(EnvProjectDir)
	}
	if explicitDir != "" {
		root, err := normalizeRoot(explicitDir)
		if err != nil {
			return nil, err
		}
		if isDir(filepath.Join(root, DirName)) {
			return &Project{Root: root, GurlDir: filepath.Join(root, DirName)}, nil
		}
		return nil, nil
	}

	if startDir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		startDir = wd
	}

	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project directory: %w", err)
	}

	for {
		gurlDir := filepath.Join(dir, DirName)
		if isDir(gurlDir) {
			return &Project{Root: dir, GurlDir: gurlDir}, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil, nil
		}
		dir = parent
	}
}

func Require(startDir string, explicitDir string) (*Project, error) {
	proj, err := Discover(startDir, explicitDir)
	if err != nil {
		return nil, err
	}
	if proj == nil {
		return nil, fmt.Errorf("gurl project not found; run 'gurl init' or set %s", EnvProjectDir)
	}
	return proj, nil
}

func Init(root string) (*Project, error) {
	if root == "" {
		root = os.Getenv(EnvProjectDir)
	}
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		root = wd
	}

	root, err := normalizeRoot(root)
	if err != nil {
		return nil, err
	}
	gurlDir := filepath.Join(root, DirName)
	proj := &Project{Root: root, GurlDir: gurlDir}

	if err := os.MkdirAll(proj.CollectionsDir(), 0755); err != nil {
		return nil, fmt.Errorf("failed to create collections directory: %w", err)
	}
	if err := os.MkdirAll(proj.EnvironmentsDir(), 0755); err != nil {
		return nil, fmt.Errorf("failed to create environments directory: %w", err)
	}
	if err := writeGitignore(filepath.Join(gurlDir, ".gitignore")); err != nil {
		return nil, err
	}

	return proj, nil
}

func (p *Project) CollectionsDir() string {
	if p == nil {
		return ""
	}
	return filepath.Join(p.GurlDir, "collections")
}

func (p *Project) EnvironmentsDir() string {
	if p == nil {
		return ""
	}
	return filepath.Join(p.GurlDir, "environments")
}

func normalizeRoot(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve project directory: %w", err)
	}
	if filepath.Base(abs) == DirName {
		return filepath.Dir(abs), nil
	}
	return abs, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func writeGitignore(path string) error {
	const contents = "# Local collection encryption keys are machine-specific.\n**/collection.key\n"
	existing, err := os.ReadFile(path)
	if err == nil {
		if string(existing) == contents {
			return nil
		}
		if len(existing) > 0 && existing[len(existing)-1] != '\n' {
			existing = append(existing, '\n')
		}
		if containsLine(string(existing), "**/collection.key") {
			return nil
		}
		existing = append(existing, []byte("**/collection.key\n")...)
		return os.WriteFile(path, existing, 0644)
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to read .gurl/.gitignore: %w", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		return fmt.Errorf("failed to write .gurl/.gitignore: %w", err)
	}
	return nil
}

func containsLine(contents string, line string) bool {
	start := 0
	for start <= len(contents) {
		end := start
		for end < len(contents) && contents[end] != '\n' {
			end++
		}
		if contents[start:end] == line {
			return true
		}
		if end == len(contents) {
			break
		}
		start = end + 1
	}
	return false
}
