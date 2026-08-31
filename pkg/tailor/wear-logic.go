package tailor

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pieroproietti/penguins-tailor/pkg/distro"
	"github.com/pieroproietti/penguins-tailor/pkg/utils"
	"gopkg.in/yaml.v3"
)

const tailorLogFile = "/var/log/tailor.log"

func logToFile(message string) {
	logPath := tailorLogFile
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		// fallback
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, message))
}

func findYaml(costumePath string) string {
	candidates := []string{
		"index.yaml",
		"index.yml",
	}

	d := distro.NewDistro()
	if d != nil {
		if d.DistroID != "" {
			candidates = append(candidates, strings.ToLower(d.DistroID)+".yaml", strings.ToLower(d.DistroID)+".yml")
		}
		if d.DistroLike != "" {
			candidates = append(candidates, strings.ToLower(d.DistroLike)+".yaml", strings.ToLower(d.DistroLike)+".yml")
		}
		if d.FamilyID != "" {
			candidates = append(candidates, strings.ToLower(d.FamilyID)+".yaml", strings.ToLower(d.FamilyID)+".yml")
		}
		if strings.EqualFold(d.FamilyID, "archlinux") {
			candidates = append(candidates, "arch.yaml", "arch.yml")
		}
		if strings.EqualFold(d.FamilyID, "debian") {
			candidates = append(candidates, "debian.yaml", "debian.yml", "ubuntu.yaml", "devuan.yaml")
		}
	}

	// Standard distro fallbacks
	candidates = append(candidates, "debian.yaml", "debian.yml", "arch.yaml", "alpine.yaml", "fedora.yaml", "opensuse.yaml")

	seen := make(map[string]struct{})
	for _, c := range candidates {
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		fullPath := filepath.Join(costumePath, c)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath
		}
	}

	return ""
}

func loadSuit(yamlFile string) (*Suit, error) {
	if yamlFile == "" {
		return nil, fmt.Errorf("costume/accessory definition yaml file not found")
	}
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return nil, err
	}
	var suit Suit
	if err := yaml.Unmarshal(data, &suit); err != nil {
		return nil, err
	}
	suit.normalize()

	// Auto-discovery: if packages.yaml or packages.yml is present in the directory,
	// automatically load and merge packages into suit.Packages
	dir := filepath.Dir(yamlFile)
	if extraPkgs := loadPackagesYaml(dir); len(extraPkgs) > 0 {
		seen := make(map[string]struct{}, len(suit.Packages)+len(extraPkgs))
		for _, p := range suit.Packages {
			seen[p] = struct{}{}
		}
		for _, p := range extraPkgs {
			if _, exists := seen[p]; !exists {
				seen[p] = struct{}{}
				suit.Packages = append(suit.Packages, p)
			}
		}
	}

	return &suit, nil
}

// loadPackagesYaml searches and loads packages.yaml or packages.yml in the specified directory
func loadPackagesYaml(dir string) []string {
	for _, name := range []string{"packages.yaml", "packages.yml"} {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		// Prova parsing con struttura standard YAML (packages: o sequence.packages:)
		var doc struct {
			Packages []string `yaml:"packages"`
			Sequence *struct {
				Packages []string `yaml:"packages"`
			} `yaml:"sequence"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil {
			var pkgs []string
			if len(doc.Packages) > 0 {
				pkgs = append(pkgs, doc.Packages...)
			}
			if doc.Sequence != nil && len(doc.Sequence.Packages) > 0 {
				pkgs = append(pkgs, doc.Sequence.Packages...)
			}
			if len(pkgs) > 0 {
				return pkgs
			}
		}

		// Prova parsing come lista semplice di stringhe (- pkg1 \n - pkg2)
		var list []string
		if err := yaml.Unmarshal(data, &list); err == nil && len(list) > 0 {
			return list
		}
	}
	return nil
}

func getAvailablePackages() map[string]struct{} {
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

// normalizePkgName strips the ":arch" multi-arch qualifier
func normalizePkgName(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.Index(name, ":"); i != -1 {
		name = name[:i]
	}
	return name
}

// isInteractiveTerminal checks if stdin is connected to a real terminal.
// This is used to decide whether to show interactive prompts or fall back
// to noninteractive mode.
func isInteractiveTerminal() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// installWithRetries installs packages, falling back to one-by-one on failure
func installWithRetries(packages []string, retries int) []string {
	return installPackagesImpl(packages, retries, false)
}

// installNoRecommends installs packages with --no-install-recommends
func installNoRecommends(packages []string) []string {
	return installPackagesImpl(packages, 3, true)
}

func installPackagesImpl(packages []string, retries int, noRecommends bool) []string {
	if len(packages) == 0 {
		return nil
	}
	if _, err := exec.LookPath("apt-get"); err != nil {
		if _, errPac := exec.LookPath("pacman"); errPac == nil {
			logToFile("Arch Linux (pacman) detected. Package installation on Arch is under development.")
			utils.LogNormal("Arch Linux (pacman) detected. Package installation on Arch is under development.\n")
			return nil
		}
		logToFile("apt-get not found on this system.")
		return packages
	}

	available := getAvailablePackages()
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
		return missing
	}

	flags := "-y"
	if noRecommends {
		flags = "-y --no-install-recommends"
	}

	// License-prompt packages must never go through the noninteractive
	// path: their preinst aborts and poisons dpkg.
	var clean []string
	for _, p := range toInstall {
		if !isLicensePrompt(p) {
			clean = append(clean, p)
		}
	}
	if len(clean) == 0 {
		return missing
	}

	// Use readline frontend if we have an interactive terminal
	debconfFrontend := "readline"
	if !isInteractiveTerminal() {
		debconfFrontend = "noninteractive"
	}

	pkgString := strings.Join(clean, " ")
	cmd := fmt.Sprintf("UCF_FORCE_CONFFOLD=1 DEBIAN_FRONTEND=%s apt-get install -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' %s %s", debconfFrontend, flags, pkgString)
	logToFile(fmt.Sprintf("Installing %d packages: %s", len(clean), pkgString))
	if err := utils.ExecTee(cmd, tailorLogFile); err == nil {
		logToFile("✅ Packages installed.")
		return missing
	}

	// Heal dpkg state before retrying
	healDpkgState()

	logToFile("⚠️  Retrying package by package to isolate failures...")
	pending := clean
	for attempt := 1; attempt <= retries && len(pending) > 0; attempt++ {
		var stillFailing []string
		for _, pkg := range pending {
			if ss := utils.GetSplitScreen(); ss != nil {
				ss.SetAction("Retrying package: %s (attempt %d/%d)", pkg, attempt, retries)
			}
			singleCmd := fmt.Sprintf("UCF_FORCE_CONFFOLD=1 DEBIAN_FRONTEND=%s apt-get install -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' %s %s", debconfFrontend, flags, pkg)
			if err := utils.ExecTee(singleCmd, tailorLogFile); err != nil {
				// Double-check with dpkg before believing the failure
				if isPackageInstalled(pkg) {
					logToFile(fmt.Sprintf("ℹ️  apt-get reported an error installing %s, but dpkg confirms it is installed correctly.", pkg))
				} else {
					stillFailing = append(stillFailing, pkg)
				}
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
	return append(missing, pending...)
}

// isPackageInstalled reports whether dpkg considers pkg to be correctly and fully installed.
func isPackageInstalled(pkg string) bool {
	out, err := exec.Command("dpkg-query", "-W", "-f=${db:Status-Status}", normalizePkgName(pkg)).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "installed"
}

// installInteractive installs packages without suppressing debconf prompts.
func installInteractive(packages []string) []string {
	if len(packages) == 0 {
		return nil
	}

	available := getAvailablePackages()
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
		return missing
	}

	// Use readline frontend for interactive packages so prompts are shown
	debconfFrontend := "readline"
	if !isInteractiveTerminal() {
		debconfFrontend = "noninteractive"
	}

	pkgString := strings.Join(toInstall, " ")
	cmd := fmt.Sprintf("DEBIAN_FRONTEND=%s apt-get install -o Dpkg::Options::='--force-confold' -y %s", debconfFrontend, pkgString)
	logToFile(fmt.Sprintf("Installing interactive packages: %s", pkgString))
	if err := utils.ExecInteractive(cmd, tailorLogFile); err != nil {
		var stillFailing []string
		for _, pkg := range toInstall {
			if !isPackageInstalled(pkg) {
				stillFailing = append(stillFailing, pkg)
			}
		}
		if len(stillFailing) > 0 {
			logToFile(fmt.Sprintf("⚠️  Some interactive packages could not be installed: %v", stillFailing))
		}
		return append(missing, stillFailing...)
	}
	return missing
}



// licensePromptPackages holds suit.PackagesInteractive: packages whose
// preinst asks a license question that cannot be answered noninteractively.
var licensePromptPackages []string

// SetLicensePromptPackages is called by Wear() before any install starts.
func SetLicensePromptPackages(pkgs []string) { licensePromptPackages = pkgs }

func isLicensePrompt(pkg string) bool {
	c := normalizePkgName(pkg)
	for _, p := range licensePromptPackages {
		if normalizePkgName(p) == c {
			return true
		}
	}
	return false
}

// healDpkgState repairs a poisoned dpkg state.
func healDpkgState() {
	// First try to configure what we can without interaction
	utils.Exec("DEBIAN_FRONTEND=noninteractive dpkg --configure -a --force-confold")
	utils.Exec("DEBIAN_FRONTEND=noninteractive apt-get install -f -y")
	
	// Then try with readline if we have a terminal
	if isInteractiveTerminal() {
		utils.ExecInteractive("DEBIAN_FRONTEND=readline dpkg --configure -a", tailorLogFile)
	}
	
	for _, p := range licensePromptPackages {
		if !isPackageInstalled(p) {
			logToFile(fmt.Sprintf("⚠️  Purging half-configured license package %s so the rest of the system can heal...", p))
			utils.Exec(fmt.Sprintf("DEBIAN_FRONTEND=noninteractive dpkg --purge --force-remove-reinstreq --force-depends %s", p))
		}
	}
	utils.Exec("DEBIAN_FRONTEND=noninteractive apt-get install -f -y")
}