package tailor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pieroproietti/penguins-tailor/pkg/utils"
)

type aptPackageManager struct{}

func (pm *aptPackageManager) Refresh() error {
	return utils.ExecLogOnly("apt-get update", tailorLogFile)
}

func (pm *aptPackageManager) Upgrade(refresh bool) error {
	if refresh {
		_ = pm.Refresh()
	}
	return utils.ExecLogOnly("UCF_FORCE_CONFFOLD=1 DEBIAN_FRONTEND=readline apt-get -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' upgrade -y", tailorLogFile)
}

func (pm *aptPackageManager) Install(packages []string, mode InstallMode) PackageInstallResult {
	if mode.Interactive {
		return pm.installInteractive(packages)
	}
	return pm.install(packages, mode.Retries, mode.NoRecommends)
}

func (pm *aptPackageManager) IsInstalled(pkg string) bool {
	out, err := exec.Command("dpkg-query", "-W", "-f=${db:Status-Status}", normalizePkgName(pkg)).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "installed"
}

// Heal preserves the recovery used after the DKMS retry phase.
func (pm *aptPackageManager) Heal() error {
	_ = utils.ExecLogOnly("dpkg --configure -a", tailorLogFile)
	_ = utils.ExecLogOnly("UCF_FORCE_CONFFOLD=1 DEBIAN_FRONTEND=readline apt-get install -f -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' -y", tailorLogFile)
	return nil
}

func (pm *aptPackageManager) install(packages []string, retries int, noRecommends bool) PackageInstallResult {
	if len(packages) == 0 {
		return PackageInstallResult{}
	}
	if _, err := exec.LookPath("apt-get"); err != nil {
		if _, errPac := exec.LookPath("pacman"); errPac == nil {
			logToFile("Arch Linux (pacman) detected. Package installation on Arch is under development.")
			utils.LogNormal("Arch Linux (pacman) detected. Package installation on Arch is under development.\n")
			return PackageInstallResult{}
		}
		logToFile("apt-get not found on this system.")
		return PackageInstallResult{}
	}

	available := pm.availablePackages()
	var toInstall []string
	var missing []string
	if available != nil {
		for _, pkg := range packages {
			cleanPkg := normalizePkgName(pkg)
			if _, ok := available[cleanPkg]; ok {
				toInstall = append(toInstall, pkg)
			} else {
				missing = append(missing, pkg)
			}
		}
	} else {
		toInstall = packages
	}

	if len(missing) > 0 {
		logToFile(fmt.Sprintf("WARNING: %d packages skipped (not found): %v", len(missing), missing))
	}
	if len(toInstall) == 0 {
		logToFile("No valid packages to install.")
		return PackageInstallResult{Unavailable: missing}
	}

	flags := "-y"
	if noRecommends {
		flags = "-y --no-install-recommends"
	}

	var clean []string
	for _, p := range toInstall {
		if !isLicensePrompt(p) {
			clean = append(clean, p)
		}
	}
	if len(clean) == 0 {
		return PackageInstallResult{Unavailable: missing}
	}

	debconfFrontend := "readline"
	if !isInteractiveTerminal() {
		debconfFrontend = "noninteractive"
	}

	pkgString := strings.Join(clean, " ")
	cmd := fmt.Sprintf("UCF_FORCE_CONFFOLD=1 DEBIAN_FRONTEND=%s apt-get install -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' %s %s", debconfFrontend, flags, pkgString)
	logToFile(fmt.Sprintf("Installing %d packages: %s", len(clean), pkgString))
	if err := utils.ExecLogOnly(cmd, tailorLogFile); err == nil {
		installed, failed := pm.partitionInstalled(clean)
		if len(failed) == 0 {
			logToFile("✅ Packages installed.")
		} else {
			logToFile(fmt.Sprintf("⚠️  %d packages were attempted but are not installed: %v", len(failed), failed))
		}
		return PackageInstallResult{Installed: installed, Unavailable: missing, Failed: failed}
	}

	pm.healInstallationState()

	logToFile("⚠️  Retrying package by package to isolate failures...")
	pending := clean
	for attempt := 1; attempt <= retries && len(pending) > 0; attempt++ {
		var stillFailing []string
		for _, pkg := range pending {
			if ss := utils.GetSplitScreen(); ss != nil {
				ss.SetAction("Retrying package: %s (attempt %d/%d)", pkg, attempt, retries)
			}
			singleCmd := fmt.Sprintf("UCF_FORCE_CONFFOLD=1 DEBIAN_FRONTEND=%s apt-get install -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' %s %s", debconfFrontend, flags, pkg)
			if err := utils.ExecLogOnly(singleCmd, tailorLogFile); err != nil {
				if pm.IsInstalled(pkg) {
					logToFile(fmt.Sprintf("ℹ️  apt-get reported an error installing %s, but dpkg confirms it is installed correctly.", pkg))
				}
			}
			if !pm.IsInstalled(pkg) {
				stillFailing = append(stillFailing, pkg)
			}
		}
		pending = stillFailing
		if len(pending) > 0 && attempt < retries {
			logToFile(fmt.Sprintf("⚠️  %d packages still failing after attempt %d/%d, retrying: %v", len(pending), attempt, retries, pending))
		}
	}

	if len(pending) > 0 {
		logToFile(fmt.Sprintf("⚠️  %d packages could not be installed: %v", len(pending), pending))
	} else {
		logToFile("✅ All packages installed successfully (one by one).")
	}
	installed, failed := pm.partitionInstalled(clean)
	return PackageInstallResult{Installed: installed, Unavailable: missing, Failed: failed}
}

func (pm *aptPackageManager) installInteractive(packages []string) PackageInstallResult {
	if len(packages) == 0 {
		return PackageInstallResult{}
	}

	available := pm.availablePackages()
	var toInstall []string
	var missing []string
	if available != nil {
		for _, pkg := range packages {
			cleanPkg := normalizePkgName(pkg)
			if _, ok := available[cleanPkg]; ok {
				toInstall = append(toInstall, pkg)
			} else {
				missing = append(missing, pkg)
			}
		}
	} else {
		toInstall = packages
	}

	if len(missing) > 0 {
		logToFile(fmt.Sprintf("WARNING: %d interactive packages skipped (not found): %v", len(missing), missing))
	}
	if len(toInstall) == 0 {
		return PackageInstallResult{Unavailable: missing}
	}

	debconfFrontend := "readline"
	if !isInteractiveTerminal() {
		debconfFrontend = "noninteractive"
	}

	pkgString := strings.Join(toInstall, " ")
	cmd := fmt.Sprintf("DEBIAN_FRONTEND=%s apt-get install -o Dpkg::Options::='--force-confold' -y %s", debconfFrontend, pkgString)
	logToFile(fmt.Sprintf("Installing interactive packages: %s", pkgString))
	if err := utils.ExecInteractive(cmd, tailorLogFile); err != nil {
		logToFile(fmt.Sprintf("⚠️  Interactive package installation command failed: %v", err))
	}
	installed, failed := pm.partitionInstalled(toInstall)
	if len(failed) > 0 {
		logToFile(fmt.Sprintf("⚠️  Some interactive packages could not be installed: %v", failed))
	}
	return PackageInstallResult{Installed: installed, Unavailable: missing, Failed: failed}
}

func (pm *aptPackageManager) partitionInstalled(packages []string) (installed []string, failed []string) {
	for _, pkg := range packages {
		if pm.IsInstalled(pkg) {
			installed = append(installed, pkg)
		} else {
			failed = append(failed, pkg)
		}
	}
	return installed, failed
}

func (pm *aptPackageManager) availablePackages() map[string]struct{} {
	available := make(map[string]struct{})
	if _, err := exec.LookPath("apt-cache"); err != nil {
		return nil
	}
	logToFile("Updating available packages database...")
	cmd := exec.Command("/usr/bin/apt-cache", "pkgnames")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return available
	}
	if err := cmd.Start(); err != nil {
		return available
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			available[line] = struct{}{}
		}
	}
	cmd.Wait()
	return available
}

func (pm *aptPackageManager) healInstallationState() {
	_ = utils.ExecLogOnly("DEBIAN_FRONTEND=noninteractive dpkg --configure -a --force-confold", tailorLogFile)
	_ = utils.ExecLogOnly("DEBIAN_FRONTEND=noninteractive apt-get install -f -y", tailorLogFile)

	if isInteractiveTerminal() {
		utils.ExecInteractive("DEBIAN_FRONTEND=readline dpkg --configure -a", tailorLogFile)
	}

	for _, p := range licensePromptPackages {
		if !pm.IsInstalled(p) {
			logToFile(fmt.Sprintf("⚠️  Purging half-configured license package %s so the rest of the system can heal...", p))
			_ = utils.ExecLogOnly(fmt.Sprintf("DEBIAN_FRONTEND=noninteractive dpkg --purge --force-remove-reinstreq --force-depends %s", p), tailorLogFile)
		}
	}
	_ = utils.ExecLogOnly("DEBIAN_FRONTEND=noninteractive apt-get install -f -y", tailorLogFile)
}

func normalizePkgName(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.Index(name, ":"); i != -1 {
		name = name[:i]
	}
	return name
}

func isInteractiveTerminal() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
