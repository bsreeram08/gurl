package share

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// KnownKeys pins the key fingerprints seen for each github:<user>, so a
// changed key set (a new laptop, or a taken-over account) is noticed instead
// of silently trusted. A nil *KnownKeys pins nothing.
type KnownKeys struct {
	// Update accepts a changed key set and re-pins it.
	Update bool
	// Notes describes pins added or updated by Check.
	Notes []string

	path    string
	keys    map[string][]string
	changed bool
}

// DefaultKnownKeysPath is ~/.config/gurl/known_keys.json.
func DefaultKnownKeysPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "gurl", "known_keys.json")
}

// LoadKnownKeys reads the pin file; a missing file is an empty set.
func LoadKnownKeys(path string) (*KnownKeys, error) {
	k := &KnownKeys{path: path, keys: map[string][]string{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return k, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &k.keys); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return k, nil
}

// Check compares fps with the pinned set for spec. The first sighting is
// pinned; a different set is an error unless Update is set.
func (k *KnownKeys) Check(spec string, fps []string) error {
	if k == nil {
		return nil
	}
	fps = slices.Clone(fps)
	slices.Sort(fps)
	old, ok := k.keys[spec]
	switch {
	case !ok:
		k.Notes = append(k.Notes, fmt.Sprintf("pinned %d key(s) for %s", len(fps), spec))
	case slices.Equal(old, fps):
		return nil
	case !k.Update:
		return fmt.Errorf("keys for %s changed since they were pinned (added: %s; removed: %s); if that's expected, re-run with --update-keys",
			spec, listOrNone(diff(fps, old)), listOrNone(diff(old, fps)))
	default:
		k.Notes = append(k.Notes, fmt.Sprintf("updated pinned keys for %s", spec))
	}
	k.keys[spec] = fps
	k.changed = true
	return nil
}

// Save writes the pin file if Check changed it.
func (k *KnownKeys) Save() error {
	if k == nil || !k.changed {
		return nil
	}
	data, err := json.MarshalIndent(k.keys, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(k.path), 0700); err != nil {
		return err
	}
	return os.WriteFile(k.path, data, 0600)
}

func diff(a, b []string) []string {
	var out []string
	for _, x := range a {
		if !slices.Contains(b, x) {
			out = append(out, x)
		}
	}
	return out
}

func listOrNone(s []string) string {
	if len(s) == 0 {
		return "none"
	}
	return strings.Join(s, ", ")
}
