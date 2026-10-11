package config

import (
	"os"
	"sort"
	"strings"
	"sync"
)

// The project was called Stash before it became BookwormWiki. Deployments
// still carry STASH_* variables in their .env files, compose files, and
// shell profiles; they keep working, and the server says which ones to
// rename.
const (
	envPrefix         = "BOOKWORMWIKI_"
	legacyEnvPrefix   = "STASH_"
	configFileVar     = "BOOKWORMWIKICONFIG"
	legacyConfigFile  = "STASHCONFIG"
	legacyNameForLogs = "Stash"
)

var (
	legacyMu      sync.Mutex
	legacyApplied = map[string]bool{}
)

// ApplyLegacyEnvironment copies every STASH_* variable to its BOOKWORMWIKI_*
// name when the new name is not set, so the rest of the program only reads
// the new names. A value under the new name always wins. It is safe to call
// more than once; NewFromFile calls it again after loading the .env file.
func ApplyLegacyEnvironment() {
	legacyMu.Lock()
	defer legacyMu.Unlock()
	for _, entry := range os.Environ() {
		name, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		var target string
		switch {
		case name == legacyConfigFile:
			target = configFileVar
		case strings.HasPrefix(name, legacyEnvPrefix) && len(name) > len(legacyEnvPrefix):
			target = envPrefix + strings.TrimPrefix(name, legacyEnvPrefix)
		default:
			continue
		}
		if _, exists := os.LookupEnv(target); exists {
			continue
		}
		if err := os.Setenv(target, value); err == nil {
			legacyApplied[name] = true
		}
	}
}

// LegacyEnvironmentNames lists the STASH_* variables that were used because
// their BOOKWORMWIKI_* name was unset, sorted, for a startup warning.
func LegacyEnvironmentNames() []string {
	legacyMu.Lock()
	defer legacyMu.Unlock()
	names := make([]string, 0, len(legacyApplied))
	for name := range legacyApplied {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
