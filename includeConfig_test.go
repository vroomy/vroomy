package vroomy

import (
	"slices"
	"testing"
)

func TestIncludeConfig_mergeAutoCert(t *testing.T) {
	type testcase struct {
		name      string
		included  IncludeConfig
		wantDir   string
		wantHosts []string
	}

	var tests = []testcase{
		{name: "absent settings preserve earlier values", wantDir: "original", wantHosts: []string{"original.example"}},
		{name: "hosts alone do not override", included: IncludeConfig{AutoCertHosts: []string{"ignored.example"}}, wantDir: "original", wantHosts: []string{"original.example"}},
		{name: "directory replaces both settings", included: IncludeConfig{AutoCertDir: "included", AutoCertHosts: []string{"included.example"}}, wantDir: "included", wantHosts: []string{"included.example"}},
		{name: "directory without hosts clears earlier hosts", included: IncludeConfig{AutoCertDir: "included"}, wantDir: "included"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var cfg = IncludeConfig{AutoCertDir: "original", AutoCertHosts: []string{"original.example"}}
			cfg.merge(&tc.included)
			if cfg.AutoCertDir != tc.wantDir || !slices.Equal(cfg.AutoCertHosts, tc.wantHosts) {
				t.Errorf("autocert settings = (%q, %v), want (%q, %v)", cfg.AutoCertDir, cfg.AutoCertHosts, tc.wantDir, tc.wantHosts)
			}
		})
	}
}
