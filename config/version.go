package config

import "strings"

// devVersion is reported when no build step recorded a version: `go build` or
// `go run .` in a checkout. Go's own VCS stamp (debug.ReadBuildInfo) isn't used
// as a fallback because the module path has no /v2 suffix, so Go ignores the
// v2.x tags and would report a misleading v1.0.x pseudo-version.
const devVersion = "dev"

var (
	// version is set at build time by the release and Docker builds:
	// -ldflags "-X aunefyren/poenskelisten/config.version=<tag>".
	version string

	// archiveVersion is filled in by `git archive` (which GitHub uses for its
	// "Source code" downloads) through the export-subst attribute on this file
	// in .gitattributes. In a normal checkout it stays unexpanded. Keep this the
	// only placeholder in the file: git substitutes every one it finds.
	archiveVersion = "$Format:%(describe:tags=true)$"
)

// resolveVersion reports the app version from the -ldflags value, else the git
// archive substitution, else devVersion.
func resolveVersion() string {
	if v := strings.TrimSpace(version); v != "" {
		return v
	}

	// An unexpanded placeholder still starts with "$"; no tag does.
	if v := strings.TrimSpace(archiveVersion); v != "" && !strings.HasPrefix(v, "$") {
		return v
	}

	return devVersion
}
