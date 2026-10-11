// Package wikiskill exposes the read-only files for BookwormWiki's bundled wiki skill.
package wikiskill

import (
	"embed"
	"io/fs"
)

// content contains documentation only; the skill ships no scripts or hooks.
//
//go:embed content/bookwormwiki-wiki
var content embed.FS

// Files returns the immutable embedded skill filesystem.
func Files() fs.FS {
	skillFiles, err := fs.Sub(content, "content/bookwormwiki-wiki")
	if err != nil {
		panic(err)
	}
	return skillFiles
}
