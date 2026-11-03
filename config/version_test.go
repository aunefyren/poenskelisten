package config

import "testing"

func TestResolveVersion(t *testing.T) {
	originalVersion, originalArchive := version, archiveVersion
	t.Cleanup(func() { version, archiveVersion = originalVersion, originalArchive })

	const unexpanded = "$Format:%(describe:tags=true)$"

	cases := []struct {
		name    string
		ldflags string
		archive string
		want    string
	}{
		{"ldflags wins over archive", "v2.5.0", "v2.4.0", "v2.5.0"},
		{"ldflags is trimmed", " beta-abc1234 \n", unexpanded, "beta-abc1234"},
		{"archive used without ldflags", "", "v2.4.1", "v2.4.1"},
		{"archive describe past a tag", "", "v2.4.1-3-gabc1234", "v2.4.1-3-gabc1234"},
		{"unexpanded archive falls back to dev", "", unexpanded, devVersion},
		{"empty archive falls back to dev", "", "", devVersion},
		{"whitespace-only values fall back to dev", "  ", " \n", devVersion},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			version, archiveVersion = tc.ldflags, tc.archive
			if got := resolveVersion(); got != tc.want {
				t.Errorf("resolveVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}
