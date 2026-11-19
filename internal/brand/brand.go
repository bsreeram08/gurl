// Package brand holds the product name and the fallbacks that keep
// installs from before the gurl -> valetkey rename working.
//
// Every lookup prefers the new name and falls back to the legacy one, so
// existing data, environment variables and project directories keep
// working without a migration step. `valetkey migrate` moves legacy data
// to the new locations when the user is ready.
package brand

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	// Name is the product and binary name.
	Name = "valetkey"
	// LegacyName is the name before the rename.
	LegacyName = "gurl"

	// EnvPrefix and LegacyEnvPrefix prefix every environment variable.
	EnvPrefix       = "VALETKEY_"
	LegacyEnvPrefix = "GURL_"

	// ProjectDirName is the per-project directory; LegacyProjectDirName is
	// still discovered when the new one is absent.
	ProjectDirName       = ".valetkey"
	LegacyProjectDirName = ".gurl"

	// DBFile and CookieDBFile are the database names inside DataDir.
	DBFile             = Name + ".db"
	CookieDBFile       = Name + "-cookies.db"
	LegacyDBFile       = LegacyName + ".db"
	LegacyCookieDBFile = LegacyName + "-cookies.db"
)

// Env names the environment variable for suffix, e.g. Env("AGENT") is
// VALETKEY_AGENT.
func Env(suffix string) string { return EnvPrefix + suffix }

// LegacyEnv names the pre-rename variable, e.g. GURL_AGENT.
func LegacyEnv(suffix string) string { return LegacyEnvPrefix + suffix }

// Getenv returns VALETKEY_<suffix> if set and non-empty, otherwise
// GURL_<suffix>.
func Getenv(suffix string) string {
	if v := os.Getenv(Env(suffix)); v != "" {
		return v
	}
	return os.Getenv(LegacyEnv(suffix))
}

// EnvSources lists both variable names, new first, for flag definitions.
func EnvSources(suffix string) []string {
	return []string{Env(suffix), LegacyEnv(suffix)}
}

// pick returns newPath if it exists, else legacyPath if it exists, else
// newPath. Callers that create the directory therefore create the new one,
// and a legacy install keeps using its existing data in place.
func pick(newPath, legacyPath string) string {
	if exists(newPath) {
		return newPath
	}
	if exists(legacyPath) {
		return legacyPath
	}
	return newPath
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DataDir is ~/.local/share/valetkey, or ~/.local/share/gurl for an install
// that has not migrated. The database, cookie store, secret key and
// collection keys all live here, so they always resolve together.
func DataDir(home string) string {
	return pick(NewDataDir(home), LegacyDataDir(home))
}

// NewDataDir and LegacyDataDir are the two candidate data directories.
func NewDataDir(home string) string { return filepath.Join(home, ".local", "share", Name) }
func LegacyDataDir(home string) string {
	return filepath.Join(home, ".local", "share", LegacyName)
}

// DBPath is the request database inside dataDir: valetkey.db, or gurl.db
// when only that exists. Resolving by file rather than by directory name
// means a moved directory never loses its database.
func DBPath(dataDir string) string {
	return pick(filepath.Join(dataDir, DBFile), filepath.Join(dataDir, LegacyDBFile))
}

// CookieDBPath is the cookie database inside dataDir, resolved like DBPath.
func CookieDBPath(dataDir string) string {
	return pick(filepath.Join(dataDir, CookieDBFile), filepath.Join(dataDir, LegacyCookieDBFile))
}

// ConfigDir is ~/.config/valetkey, or ~/.config/gurl when only that exists.
func ConfigDir(home string) string {
	return pick(NewConfigDir(home), LegacyConfigDir(home))
}

func NewConfigDir(home string) string    { return filepath.Join(home, ".config", Name) }
func LegacyConfigDir(home string) string { return filepath.Join(home, ".config", LegacyName) }

// PluginDir is ~/.valetkey/plugins, or ~/.gurl/plugins when only that exists.
func PluginDir(home string) string {
	return pick(NewPluginDir(home), LegacyPluginDir(home))
}

func NewPluginDir(home string) string { return filepath.Join(home, "."+Name, "plugins") }
func LegacyPluginDir(home string) string {
	return filepath.Join(home, "."+LegacyName, "plugins")
}

// RCFiles are the dotfile config names, new first.
var RCFiles = []string{"." + Name + "rc", "." + LegacyName + "rc"}

// BinaryNames are the program names the binary answers to; AgentBinaryNames
// turn on agent mode.
var (
	BinaryNames      = []string{Name, LegacyName}
	AgentBinaryNames = []string{"agent" + Name, "agent" + LegacyName}
)

// IsAgentBinary reports whether arg0 is one of AgentBinaryNames (with or
// without .exe).
func IsAgentBinary(arg0 string) bool {
	name := filepath.Base(arg0)
	if ext := filepath.Ext(name); strings.EqualFold(ext, ".exe") {
		name = strings.TrimSuffix(name, ext)
	}
	for _, n := range AgentBinaryNames {
		if strings.EqualFold(name, n) {
			return true
		}
	}
	return false
}
