package git

import (
	"strings"
	"testing"
)

func TestParseGitVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    Version
		wantErr bool
	}{
		{
			name:  "standard",
			input: "git version 2.39.0",
			want:  Version{2, 39, 0},
		},
		{
			name:  "apple suffix",
			input: "git version 2.39.3 (Apple Git-146)",
			want:  Version{2, 39, 3},
		},
		{
			name:  "windows suffix",
			input: "git version 2.42.0.windows.1",
			want:  Version{2, 42, 0},
		},
		{
			name:  "two components",
			input: "git version 2.9",
			want:  Version{2, 9, 0},
		},
		{
			name:  "old version",
			input: "git version 1.8.5",
			want:  Version{1, 8, 5},
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "garbage",
			input:   "not a version string",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGitVersion(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseGitVersion(%q) expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseGitVersion(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseGitVersion(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestVersionAtLeast(t *testing.T) {
	tests := []struct {
		name string
		v    Version
		min  Version
		want bool
	}{
		{"equal", Version{2, 38, 0}, Version{2, 38, 0}, true},
		{"greater major", Version{3, 0, 0}, Version{2, 38, 0}, true},
		{"greater minor", Version{2, 39, 0}, Version{2, 38, 0}, true},
		{"greater patch", Version{2, 38, 1}, Version{2, 38, 0}, true},
		{"less major", Version{1, 99, 99}, Version{2, 38, 0}, false},
		{"less minor", Version{2, 37, 99}, Version{2, 38, 0}, false},
		{"less patch", Version{2, 38, 0}, Version{2, 38, 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.v.AtLeast(tt.min)
			if got != tt.want {
				t.Errorf("%v.AtLeast(%v) = %v, want %v", tt.v, tt.min, got, tt.want)
			}
		})
	}
}

func TestInstallHint(t *testing.T) {
	tests := []struct {
		goos         string
		wantContains string
	}{
		{"darwin", "brew install git"},
		{"darwin", "xcode-select --install"},
		{"linux", "apt-get install git"},
		{"linux", "dnf install git"},
		{"linux", "pacman -S git"},
		{"windows", "winget install Git.Git"},
		{"windows", "git-scm.com"},
		{"freebsd", "git-scm.com"},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"_"+tt.wantContains, func(t *testing.T) {
			hint := InstallHint(tt.goos)
			if !strings.Contains(hint, tt.wantContains) {
				t.Errorf("InstallHint(%q) = %q, want it to contain %q", tt.goos, hint, tt.wantContains)
			}
		})
	}
}

func TestVersionString(t *testing.T) {
	v := Version{2, 38, 0}
	if s := v.String(); s != "2.38.0" {
		t.Errorf("Version.String() = %q, want %q", s, "2.38.0")
	}
}
