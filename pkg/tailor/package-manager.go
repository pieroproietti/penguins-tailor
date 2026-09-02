package tailor

// InstallMode controls package installation behavior requested by a costume.
type InstallMode struct {
	NoRecommends bool
	Interactive  bool
	Retries      int
}

// PackageInstallResult describes the final outcome of packages requested from
// a package manager. Unavailable packages are not attempted; failed packages
// were attempted but are still not installed after the operation completes.
type PackageInstallResult struct {
	Installed   []string
	Unavailable []string
	Failed      []string
}

func (r *PackageInstallResult) merge(other PackageInstallResult) {
	r.Installed = append(r.Installed, other.Installed...)
	r.Unavailable = append(r.Unavailable, other.Unavailable...)
	r.Failed = append(r.Failed, other.Failed...)
}

// PackageManager performs package operations required by tailor wear.
type PackageManager interface {
	Refresh() error
	Upgrade(refresh bool) error
	Install(packages []string, mode InstallMode) PackageInstallResult
	IsInstalled(pkg string) bool
	Heal() error
}

// newPackageManager returns the current package manager implementation.
// The APT implementation retains the existing fallback behavior when APT is
// unavailable, including the Arch/Manjaro "under development" message.
func newPackageManager() PackageManager {
	return &aptPackageManager{}
}
