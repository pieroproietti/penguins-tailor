package tailor

// InstallMode controls package installation behavior requested by a costume.
type InstallMode struct {
	NoRecommends bool
	Interactive  bool
	Retries      int
}

// PackageManager performs package operations required by tailor wear.
type PackageManager interface {
	Refresh() error
	Upgrade() error
	Install(packages []string, mode InstallMode) []string
	IsInstalled(pkg string) bool
	Heal() error
}

// newPackageManager returns the current package manager implementation.
// The APT implementation retains the existing fallback behavior when APT is
// unavailable, including the Arch/Manjaro "under development" message.
func newPackageManager() PackageManager {
	return &aptPackageManager{}
}
