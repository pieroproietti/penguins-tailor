package distro

import "testing"

func TestIdentityUsesCodenameAndReleaseFallback(t *testing.T) {
	tests := []struct {
		name string
		d    Distro
		want string
	}{
		{"codename", Distro{DistroID: "Debian", CodenameID: "Bookworm", ReleaseID: "12"}, "debian-bookworm"},
		{"release fallback", Distro{DistroID: "Arch Linux", ReleaseID: "rolling release"}, "arch-linux-rolling-release"},
		{"distribution only", Distro{DistroID: "Alpine"}, "alpine"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.Identity(); got != tt.want {
				t.Fatalf("Identity() = %q, want %q", got, tt.want)
			}
		})
	}
}
