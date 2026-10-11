// Package workskill exposes the read-only files for BookwormWiki's bundled work skill.
package workskill

import (
	"embed"
	"io/fs"
)

// content contains documentation only. In particular, the skill does not ship
// scripts, hooks, or any other host-side execution surface.
//
//go:embed content/bookwormwiki-work
var content embed.FS

// Files returns the immutable embedded skill filesystem.
func Files() fs.FS {
	skillFiles, err := fs.Sub(content, "content/bookwormwiki-work")
	if err != nil {
		panic(err)
	}
	return skillFiles
}
