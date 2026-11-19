package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bsreeram08/valetkey/internal/brand"
)

const (
	// DirName is the project directory. LegacyDirName (.gurl) is still
	// discovered when a root has no DirName.
	DirName       = brand.ProjectDirName
	LegacyDirName = brand.LegacyProjectDirName
	// EnvProjectDir names the project root; GURL_PROJECT_DIR still works.
	EnvProjectDir = brand.EnvPrefix + "PROJECT_DIR"
)

// Project is a discovered project: Root holds Dir (.valetkey or .gurl).
type Project struct {
	Root string
	Dir  string
}

// projectDirIn returns the project directory inside root, preferring
// .valetkey over .gurl, or "" when root has neither.
func projectDirIn(root string) string {
	for _, name := range []string{DirName, LegacyDirName} {
		if dir := filepath.Join(root, name); isDir(dir) {
			return dir
		}
	}
	return ""
}

func Discover(startDir string, explicitDir string) (*Project, error) {
	if explicitDir == "" {
		explicitDir = brand.Getenv("PROJECT_DIR")
	}
	if explicitDir != "" {
		root, err := normalizeRoot(explicitDir)
		if err != nil {
			return nil, err
		}
		if dir := projectDirIn(root); dir != "" {
			return &Project{Root: root, Dir: dir}, nil
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
		if projDir := projectDirIn(dir); projDir != "" {
			return &Project{Root: dir, Dir: projDir}, nil
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
		return nil, fmt.Errorf("%s project not found; run '%s init' or set %s", brand.Name, brand.Name, EnvProjectDir)
	}
	return proj, nil
}

func Init(root string) (*Project, error) {
	if root == "" {
		root = brand.Getenv("PROJECT_DIR")
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
	// Re-running init in a project that still uses .gurl keeps using it
	// rather than starting a second, empty .valetkey next to it.
	projDir := projectDirIn(root)
	if projDir == "" {
		projDir = filepath.Join(root, DirName)
	}
	proj := &Project{Root: root, Dir: projDir}

	if err := os.MkdirAll(proj.CollectionsDir(), 0755); err != nil {
		return nil, fmt.Errorf("failed to create collections directory: %w", err)
	}
	if err := os.MkdirAll(proj.EnvironmentsDir(), 0755); err != nil {
		return nil, fmt.Errorf("failed to create environments directory: %w", err)
	}
	if err := writeGitignore(filepath.Join(projDir, ".gitignore")); err != nil {
		return nil, err
	}

	return proj, nil
}

func (p *Project) CollectionsDir() string {
	if p == nil {
		return ""
	}
	return filepath.Join(p.Dir, "collections")
}

func (p *Project) EnvironmentsDir() string {
	if p == nil {
		return ""
	}
	return filepath.Join(p.Dir, "environments")
}

func normalizeRoot(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve project directory: %w", err)
	}
	if base := filepath.Base(abs); base == DirName || base == LegacyDirName {
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
		return fmt.Errorf("failed to read project .gitignore: %w", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		return fmt.Errorf("failed to write project .gitignore: %w", err)
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
