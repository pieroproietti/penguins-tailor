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
)

func Wear(costumeName string, noAcc bool, noFirm bool, linear bool, branch string, dryRun bool) error {
	d := distro.NewDistro()
	if d.FamilyID != "debian" && d.FamilyID != "archlinux" {
		utils.LogError("Distribution '%s' (family: %s) is not supported. Tailor currently supports Debian and Arch derivatives.", d.DistroID, d.FamilyID)
		return fmt.Errorf("unsupported distribution family: %s", d.FamilyID)
	}

	if os.Geteuid() != 0 && !dryRun {
		utils.LogError("'tailor wear' needs to install packages and write to system paths; run it as root (e.g. 'sudo tailor wear %s').", costumeName)
		return fmt.Errorf("must be run as root")
	}

	root, err := getWardrobeRoot()
	if err != nil {
		utils.LogError("Wardrobe root error: %v", err)
		return err
	}

	// If branch is specified, or if the wardrobe repository does not exist yet (and we're not in local ./v2 dev mode),
	// ensure wardrobe is fetched/cloned and on the right branch.
	if branch != "" {
		if err := Get("", branch); err != nil {
			return fmt.Errorf("failed to get costumes repository on branch '%s': %w", branch, err)
		}
	} else if _, errStat := os.Stat(root); os.IsNotExist(errStat) {
		if stat, errV2 := os.Stat("v2"); errV2 != nil || !stat.IsDir() {
			if err := Get("", ""); err != nil {
				return fmt.Errorf("failed to download costumes repository: %w", err)
			}
		}
	}

	v2Dir, err := getWardrobeV2Dir()
	if err != nil {
		utils.LogError("Wardrobe root error: %v", err)
		return err
	}
	costumeDir := filepath.Join(v2Dir, "costumes", costumeName)
	if _, err := os.Stat(costumeDir); os.IsNotExist(err) {
		if strings.HasPrefix(costumeName, "accessories/") || strings.HasPrefix(costumeName, "costumes/") {
			costumeDir = filepath.Join(v2Dir, costumeName)
		} else {
			accDir := filepath.Join(v2Dir, "accessories", costumeName)
			if _, errAcc := os.Stat(accDir); errAcc == nil {
				costumeDir = accDir
			}
		}
	}
	if _, err := os.Stat(costumeDir); os.IsNotExist(err) {
		return fmt.Errorf("costume '%s' not found in %s", costumeName, costumeDir)
	}

	yamlFile := findYaml(costumeDir)
	suit, err := loadSuit(yamlFile)
	if err != nil {
		return err
	}

	// Enforce distribution compatibility BEFORE anything else
	if err := checkCostumeCompatibility(costumeDir, suit); err != nil {
		utils.LogError("%s", incompatibleDistroMessage(suit.Name, suit.Distributions, currentDistroName()))
		return err
	}

	isDirectAccessory := strings.HasPrefix(costumeName, "accessories/") || (suit.Name != "" && !strings.Contains(costumeDir, "/costumes/"))

	origin := GetWardrobeOrigin()
	activeBranch := GetWardrobeBranch()

	costumeLabel := fmt.Sprintf("Costume: %s", suit.Name)
	if suit.Release != "" {
		costumeLabel = fmt.Sprintf("Costume: %s (v%s)", suit.Name, suit.Release)
	}
	if isDirectAccessory {
		costumeLabel = fmt.Sprintf("Accessory: %s", suit.Name)
		if suit.Release != "" {
			costumeLabel = fmt.Sprintf("Accessory: %s (v%s)", suit.Name, suit.Release)
		}
	}

	notes := suit.Description
	if findPreseed(costumeDir) != "" {
		if notes != "" {
			notes += " - Preseed applied"
		} else {
			notes = "Preseed applied"
		}
	}
	if dryRun {
		if notes != "" {
			notes = "[DRY-RUN] " + notes
		} else {
			notes = "[DRY-RUN] Simulation mode (no changes will be applied)"
		}
	}

	headerCfg := utils.SplitScreenConfig{
		Atelier: origin,
		Costume: costumeLabel,
		Branch:  activeBranch,
		Notes:   notes,
	}

	var ss *utils.SplitScreen
	if !linear {
		ss = utils.StartSplitScreenConfig(headerCfg)
		if ss != nil {
			defer ss.Close()
		}
	}
	if ss == nil {
		utils.PrintBannerConfig(headerCfg)
	}

	// DKMS safety: ensure headers for running kernel are present
	if ss != nil {
		ss.SetAction("Checking kernel headers for DKMS...")
		if err := ensureKernelHeaders(dryRun); err != nil {
			ss.AddStep(fmt.Sprintf("%s[WARN]%s Kernel headers verification completed with warnings", utils.ColorYellow, utils.ColorReset))
		} else {
			statusMsg := "Kernel headers verified"
			if dryRun {
				statusMsg += " (simulated)"
			}
			ss.AddStep(fmt.Sprintf("%s[OK]%s %s", utils.ColorGreen, utils.ColorReset, statusMsg))
		}
	} else {
		spHeaders := utils.NewSpinner("Checking kernel headers for DKMS...")
		spHeaders.Start()
		if err := ensureKernelHeaders(dryRun); err != nil {
			spHeaders.Warn("Kernel headers verification completed with warnings")
		} else {
			statusMsg := "Kernel headers verified"
			if dryRun {
				statusMsg += " (simulated)"
			}
			spHeaders.Success("%s", statusMsg)
		}
	}

	SetLicensePromptPackages(suit.PackagesInteractive)

	installedPackages, failedPackages, err := applySuit(costumeDir, suit, dryRun, false)
	if err != nil {
		return err
	}

	if !noAcc && len(suit.Accessories) > 0 {
		if ss == nil {
			utils.PrintSection("👝", fmt.Sprintf("ACCESSORIES (%d items)", len(suit.Accessories)))
		}
		for idx, accName := range suit.Accessories {
			if noFirm && (accName == "firmwares" || strings.Contains(accName, "firmware")) {
				if ss != nil {
					ss.AddStep(fmt.Sprintf("%s[INFO]%s Skipping firmware accessory '%s' (--no-firm)", utils.ColorYellow, utils.ColorReset, accName))
				} else {
					fmt.Printf("\n  %s[INFO] Skipping firmware accessory '%s' due to --no-firm flag%s\n", utils.ColorYellow, accName, utils.ColorReset)
				}
				continue
			}

			var accDir string
			if strings.HasPrefix(accName, "./") || strings.HasPrefix(accName, "../") {
				accDir = filepath.Join(costumeDir, accName)
			} else if strings.HasPrefix(accName, "accessories/") {
				accDir = filepath.Join(v2Dir, accName)
			} else {
				accDir = filepath.Join(v2Dir, "accessories", accName)
			}

			if accYaml := findYaml(accDir); accYaml != "" {
				if accSuit, err := loadSuit(accYaml); err == nil {
					if ss != nil {
						ss.SetAction("Accessory [%d/%d]: %s...", idx+1, len(suit.Accessories), accName)
					}
					accInstalled, accFailed, _ := applySuit(accDir, accSuit, dryRun, true)
					installedPackages = append(installedPackages, accInstalled...)
					failedPackages = append(failedPackages, accFailed...)

					// If the accessory defines nested accessories, apply them recursively
					if len(accSuit.Accessories) > 0 {
						for subIdx, subAccName := range accSuit.Accessories {
							if noFirm && (subAccName == "firmwares" || strings.Contains(subAccName, "firmware")) {
								continue
							}
							var subAccDir string
							if strings.HasPrefix(subAccName, "./") || strings.HasPrefix(subAccName, "../") {
								subAccDir = filepath.Join(accDir, subAccName)
							} else if strings.HasPrefix(subAccName, "accessories/") {
								subAccDir = filepath.Join(v2Dir, subAccName)
							} else {
								subAccDir = filepath.Join(v2Dir, "accessories", subAccName)
							}
							if subAccYaml := findYaml(subAccDir); subAccYaml != "" {
								if subAccSuit, err := loadSuit(subAccYaml); err == nil {
									if ss != nil {
										ss.SetAction("  Nested accessory [%d/%d]: %s...", subIdx+1, len(accSuit.Accessories), subAccName)
									}
									subInstalled, subFailed, _ := applySuit(subAccDir, subAccSuit, dryRun, true)
									installedPackages = append(installedPackages, subInstalled...)
									failedPackages = append(failedPackages, subFailed...)
									if len(subAccSuit.FinalizeCmds) > 0 {
										executeFinalizeCommands(subAccSuit.FinalizeCmds, subAccDir, subAccSuit.Name, dryRun)
									}
								}
							}
						}
					}

					// If the accessory defines its own FinalizeCmds, execute them for this accessory
					if len(accSuit.FinalizeCmds) > 0 {
						executeFinalizeCommands(accSuit.FinalizeCmds, accDir, accSuit.Name, dryRun)
					}

					accPreseedSuffix := ""
					if findPreseed(accDir) != "" {
						accPreseedSuffix = " - Preseed applied"
					}
					accDetails := ""
					if len(accInstalled) > 0 || len(accFailed) > 0 {
						if len(accFailed) > 0 {
							accDetails = fmt.Sprintf(" - Installed %d packages (%d failed)", len(accInstalled), len(accFailed))
						} else {
							suffix := ""
							if dryRun {
								suffix = " (simulated)"
							}
							accDetails = fmt.Sprintf(" - Installed %d packages%s", len(accInstalled), suffix)
						}
					}

					if ss != nil {
						color := utils.ColorCyan
						if len(accFailed) > 0 {
							color = utils.ColorYellow
						}
						ss.AddStep(fmt.Sprintf("%s--> [%d/%d] Accessory: %s%s%s%s", color, idx+1, len(suit.Accessories), accName, accDetails, accPreseedSuffix, utils.ColorReset))
					} else {
						utils.PrintSubSection("-->", fmt.Sprintf("[%d/%d] Accessory: %s%s%s", idx+1, len(suit.Accessories), accName, accDetails, accPreseedSuffix))
					}
				} else {
					if ss != nil {
						ss.AddStep(fmt.Sprintf("%s[WARN]%s Could not load accessory '%s'", utils.ColorYellow, utils.ColorReset, accName))
					} else {
						fmt.Printf("  %s[WARN] Could not load accessory '%s': %v%s\n", utils.ColorYellow, accName, err, utils.ColorReset)
					}
				}
			} else {
				if ss != nil {
					ss.AddStep(fmt.Sprintf("%s[WARN]%s Accessory '%s' not found", utils.ColorYellow, utils.ColorReset, accName))
				} else {
					fmt.Printf("  %s[WARN] Accessory '%s' not found in %s%s\n", utils.ColorYellow, accName, accDir, utils.ColorReset)
				}
			}
		}
	}

	// DKMS healing
	if !dryRun {
		if ss != nil {
			ss.SetAction("Healing DKMS state...")
		}
		failedPackages = healAndRetryFailed(failedPackages)
	}

	// Costume Sysroot Overlay
	applySysroot(costumeDir, suit.Name, dryRun, false)

	// Costume Finalization commands
	if len(suit.FinalizeCmds) > 0 {
		executeFinalizeCommands(suit.FinalizeCmds, costumeDir, suit.Name, dryRun)
	}

	// User environment synchronization
	targetUser := getTargetUsername()
	if targetUser != "" && targetUser != "root" {
		if ss != nil {
			ss.SetAction("Synchronizing user environment (/etc/skel -> /home/%s)...", targetUser)
		}
		copySkelToUser(dryRun)
		if ss != nil {
			statusMsg := fmt.Sprintf("User environment synchronized (%s)", targetUser)
			if dryRun {
				statusMsg += " (simulated)"
			}
			ss.AddStep(fmt.Sprintf("%s[OK]%s %s", utils.ColorGreen, utils.ColorReset, statusMsg))
		}
	}

	// Prompt user to review terminal output before closing split screen and showing summary
	waitKeyPress("Press Enter to continue to summary report...")

	// Close split screen before printing final summary box
	if ss != nil {
		ss.Close()
		ss = nil
	}

	reportPath, reportErr := writeWearReport(wearReport{
		CostumeName:   suit.Name,
		Installed:     installedPackages,
		FailedInstall: failedPackages,
	})

	costumeSummaryVal := suit.Name
	if dryRun {
		costumeSummaryVal = fmt.Sprintf("%s [DRY-RUN]", suit.Name)
	}
	summaryRows := [][2]string{
		{"Costume / Item", costumeSummaryVal},
	}
	if origin != "" {
		atelierVal := origin
		if activeBranch != "" && activeBranch != "main" && activeBranch != "master" {
			atelierVal = fmt.Sprintf("%s (%s)", origin, activeBranch)
		}
		summaryRows = append(summaryRows, [2]string{"Atelier", atelierVal})
	}
	pkgInstalledLabel := "Packages installed"
	if dryRun {
		pkgInstalledLabel = "Packages to install"
	}
	summaryRows = append(summaryRows,
		[2]string{pkgInstalledLabel, fmt.Sprintf("%d", len(installedPackages))},
		[2]string{"Packages NOT installed", fmt.Sprintf("%d", len(failedPackages))},
	)
	if reportErr == nil {
		summaryRows = append(summaryRows, [2]string{"Detailed report", reportPath})
	}
	if !dryRun {
		summaryRows = append(summaryRows, [2]string{"System log", tailorLogFile})
	}

	summaryTitle := "✨ WEAR COMPLETED!"
	if dryRun {
		summaryTitle = "✨ WEAR COMPLETED (SIMULATION / DRY-RUN)!"
	}
	utils.PrintSummaryBox(summaryTitle, summaryRows)

	if !dryRun {
		printKernelCleanupReminder()
		if suit.DisplayManagerNotice {
			printDisplayManagerNotice()
		}
		if suit.Reboot {
			rebootSystem()
		}
	}
	return nil
}

func getTargetUsername() string {
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		return sudoUser
	}
	if u := firstHumanUser(); u != nil {
		return u.Username
	}
	return ""
}

// checkCostumeCompatibility enforces the distribution compatibility declared by the costume.
func checkCostumeCompatibility(costumeDir string, suit *Suit) error {
	if len(suit.Distributions) == 0 {
		return nil
	}

	current := currentDistroCodename()
	if current == "" {
		logToFile("WARNING: could not detect the running distribution; skipping compatibility check.")
		return nil
	}

	for _, d := range suit.Distributions {
		if strings.EqualFold(strings.TrimSpace(d), current) {
			return nil
		}
	}

	script := filepath.Join(costumeDir, "tailor-check")
	if _, err := os.Stat(script); os.IsNotExist(err) {
		script = filepath.Join(costumeDir, "wardrobe-check")
	}
	if _, err := os.Stat(script); err == nil {
		cmd := exec.Command("bash", script)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	} else {
		fmt.Fprintf(os.Stderr,
			"This costume (%s) is only compatible with: %s. Detected distribution: %s.\n",
			suit.Name, strings.Join(suit.Distributions, ", "), current)
	}

	return fmt.Errorf("aborted: distribution %q not supported", current)
}

// currentDistroCodename reads /etc/os-release and returns the VERSION_CODENAME
func currentDistroCodename() string {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "VERSION_CODENAME=") {
			return strings.Trim(strings.TrimPrefix(line, "VERSION_CODENAME="), `"`)
		}
	}
	return ""
}

// ensureKernelHeaders installs the kernel headers matching the currently running kernel.
func ensureKernelHeaders(dryRun bool) error {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		logToFile(fmt.Sprintf("WARNING: could not determine running kernel version: %v", err))
		return err
	}
	release := strings.TrimSpace(string(out))
	if release == "" {
		return nil
	}
	archOut, _ := exec.Command("dpkg", "--print-architecture").Output()
	arch := strings.TrimSpace(string(archOut))
	if arch == "" {
		arch = "amd64"
	}
	pkgs := fmt.Sprintf("linux-headers-%s linux-headers-%s", release, arch)
	if dryRun {
		logToFile(fmt.Sprintf("[DRY-RUN] Would ensure kernel headers are present: %s", pkgs))
		return nil
	}
	logToFile(fmt.Sprintf("Ensuring kernel headers are present before DKMS installs: %s", pkgs))
	return utils.ExecTee("UCF_FORCE_CONFFOLD=1 DEBIAN_FRONTEND=readline apt-get install -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' -y "+pkgs, tailorLogFile)
}

// healAndRetryFailed repairs the half-configured dpkg state and retries failed packages.
func healAndRetryFailed(failed []string) []string {
	if len(failed) == 0 {
		return nil
	}

	logToFile("Healing dpkg state before retrying failed packages...")
	_ = utils.ExecTee("dpkg --configure -a", tailorLogFile)
	_ = utils.ExecTee("UCF_FORCE_CONFFOLD=1 DEBIAN_FRONTEND=readline apt-get install -f -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' -y", tailorLogFile)

	available := getAvailablePackages()
	var retry []string
	for _, p := range failed {
		if available == nil {
			retry = append(retry, p)
			continue
		}
		if _, ok := available[normalizePkgName(p)]; ok {
			retry = append(retry, p)
		}
	}
	if len(retry) == 0 {
		return failed
	}

	logToFile(fmt.Sprintf("Retrying %d packages now that kernel headers are in place...", len(retry)))
	installWithRetries(retry, 1)

	var still []string
	for _, p := range failed {
		if !isPackageInstalled(p) {
			still = append(still, p)
		}
	}
	return still
}

// applySuit applies a costume or accessory definition with clean spinners
func applySuit(dir string, suit *Suit, dryRun bool, isAccessory bool) ([]string, []string, error) {
	var installedPackages []string
	var failedPackages []string
	ss := utils.GetSplitScreen()

	// Preseed (debconf-set-selections)
	if preseedFile := findPreseed(dir); preseedFile != "" {
		if ss != nil {
			ss.SetAction("Applying preseed selections (%s)...", filepath.Base(preseedFile))
		}
		if dryRun {
			logToFile(fmt.Sprintf("[DRY-RUN] Would apply preseed: %s", preseedFile))
		} else {
			if err := applyPreseed(preseedFile, suit.Name); err != nil {
				logToFile(WarnPrefix(suit.Name) + fmt.Sprintf("Preseed application warning: %v", err))
			}
		}
	}

	// Repositories
	if suit.Sequence != nil && suit.Sequence.Repositories != nil {
		if ss != nil {
			if suit.Name != "" {
				ss.SetAction("%s, configuring package repositories & updating cache...", suit.Name)
			} else {
				ss.SetAction("Configuring package repositories & updating cache...")
			}
		}
		setupRepositories(suit.Sequence.Repositories, suit.Name, dryRun)
		if !isAccessory && ss != nil {
			statusMsg := "Repositories configured & updated"
			if dryRun {
				statusMsg += " (simulated)"
			}
			ss.AddStep(fmt.Sprintf("%s[OK]%s %s", utils.ColorGreen, utils.ColorReset, statusMsg))
		}
	}

	// Packages
	if len(suit.Packages) > 0 {
		if ss != nil {
			if suit.Name != "" {
				ss.SetAction("%s, installing packages (%d packages)...", suit.Name, len(suit.Packages))
			} else {
				ss.SetAction("Installing packages (%d packages)...", len(suit.Packages))
			}
		}
		var failed []string
		if dryRun {
			logToFile(fmt.Sprintf("[DRY-RUN] Would install %d packages: %v", len(suit.Packages), suit.Packages))
			installedPackages = append(installedPackages, suit.Packages...)
		} else {
			failed = installWithRetries(suit.Packages, 3)
			failedPackages = append(failedPackages, failed...)
			installed := diffStr(suit.Packages, failed)
			installedPackages = append(installedPackages, installed...)
		}
		if !isAccessory && ss != nil {
			if len(failed) > 0 {
				ss.AddStep(fmt.Sprintf("%s[WARN]%s Installed %d packages (%d failed)", utils.ColorYellow, utils.ColorReset, len(suit.Packages)-len(failed), len(failed)))
			} else {
				suffix := ""
				if dryRun {
					suffix = " (simulated)"
				}
				ss.AddStep(fmt.Sprintf("%s[OK]%s Installed %d packages%s", utils.ColorGreen, utils.ColorReset, len(suit.Packages), suffix))
			}
		}
	}

	// Packages No Recommends
	if len(suit.PackagesNoRecommends) > 0 {
		if ss != nil {
			if suit.Name != "" {
				ss.SetAction("%s, installing packages without recommends (%d packages)...", suit.Name, len(suit.PackagesNoRecommends))
			} else {
				ss.SetAction("Installing packages without recommends (%d packages)...", len(suit.PackagesNoRecommends))
			}
		}
		var failed []string
		if dryRun {
			logToFile(fmt.Sprintf("[DRY-RUN] Would install %d packages without recommends: %v", len(suit.PackagesNoRecommends), suit.PackagesNoRecommends))
			installedPackages = append(installedPackages, suit.PackagesNoRecommends...)
		} else {
			failed = installNoRecommends(suit.PackagesNoRecommends)
			failedPackages = append(failedPackages, failed...)
			installed := diffStr(suit.PackagesNoRecommends, failed)
			installedPackages = append(installedPackages, installed...)
		}
		if !isAccessory && ss != nil {
			if len(failed) > 0 {
				ss.AddStep(fmt.Sprintf("%s[WARN]%s Installed %d packages without recommends (%d failed)", utils.ColorYellow, utils.ColorReset, len(suit.PackagesNoRecommends)-len(failed), len(failed)))
			} else {
				suffix := ""
				if dryRun {
					suffix = " (simulated)"
				}
				ss.AddStep(fmt.Sprintf("%s[OK]%s Installed %d packages without recommends%s", utils.ColorGreen, utils.ColorReset, len(suit.PackagesNoRecommends), suffix))
			}
		}
	}

	// Packages Interactive
	if len(suit.PackagesInteractive) > 0 {
		if ss != nil {
			if suit.Name != "" {
				ss.SetAction("%s, configuring %d interactive packages...", suit.Name, len(suit.PackagesInteractive))
			} else {
				ss.SetAction("Configuring %d interactive packages...", len(suit.PackagesInteractive))
			}
		}
		var failed []string
		if dryRun {
			logToFile(fmt.Sprintf("[DRY-RUN] Would configure %d interactive packages: %v", len(suit.PackagesInteractive), suit.PackagesInteractive))
			installedPackages = append(installedPackages, suit.PackagesInteractive...)
		} else {
			failed = installInteractive(suit.PackagesInteractive)
			failedPackages = append(failedPackages, failed...)
			installed := diffStr(suit.PackagesInteractive, failed)
			installedPackages = append(installedPackages, installed...)
		}
		if !isAccessory && ss != nil {
			if len(failed) > 0 {
				ss.AddStep(fmt.Sprintf("%s[WARN]%s Some interactive packages could not be installed", utils.ColorYellow, utils.ColorReset))
			} else {
				suffix := ""
				if dryRun {
					suffix = " (simulated)"
				}
				ss.AddStep(fmt.Sprintf("%s[OK]%s Interactive packages configured%s", utils.ColorGreen, utils.ColorReset, suffix))
			}
		}
	}

	// For accessories, apply their sysroot overlay right after their packages
	if isAccessory {
		applySysroot(dir, suit.Name, dryRun, true)
	}

	// Sequence commands (intermediate commands defined in sequence.cmds)
	if len(suit.SequenceCmds) > 0 {
		if ss != nil {
			if suit.Name != "" {
				ss.SetAction("%s, running sequence scripts (%d commands)...", suit.Name, len(suit.SequenceCmds))
			} else {
				ss.SetAction("Running sequence scripts (%d commands)...", len(suit.SequenceCmds))
			}
		}
		for idx, command := range suit.SequenceCmds {
			fields := strings.Fields(command)
			cmdName := command
			if len(fields) > 0 {
				cmdName = filepath.Base(fields[0])
			}
			if ss != nil {
				if suit.Name != "" {
					ss.SetAction("%s, [%d/%d] Sequence: %s", suit.Name, idx+1, len(suit.SequenceCmds), cmdName)
				} else {
					ss.SetAction("[%d/%d] Sequence: %s", idx+1, len(suit.SequenceCmds), cmdName)
				}
			}
			if dryRun {
				logToFile(fmt.Sprintf("[DRY-RUN] Would run sequence command: %s", command))
				continue
			}
			if len(fields) > 0 {
				relScript := filepath.Join(dir, fields[0])
				if stat, err := os.Stat(relScript); err == nil && !stat.IsDir() {
					rest := strings.TrimSpace(command[len(fields[0]):])
					var fullCmd string
					if rest != "" {
						fullCmd = fmt.Sprintf("%s %s", relScript, rest)
					} else {
						fullCmd = fmt.Sprintf("%s %s", relScript, suit.Name)
					}
					_ = utils.ExecTee(fullCmd, tailorLogFile)
					continue
				}
			}
			_ = utils.ExecTee(command, tailorLogFile)
		}
		if !isAccessory && ss != nil {
			suffix := ""
			if dryRun {
				suffix = " (simulated)"
			}
			ss.AddStep(fmt.Sprintf("%s[OK]%s Sequence commands completed%s", utils.ColorGreen, utils.ColorReset, suffix))
		}
	}

	return installedPackages, failedPackages, nil
}

// applySysroot applies the sysroot/ or dirs/ filesystem overlay
func applySysroot(dir string, suitName string, dryRun bool, isAccessory bool) {
	ss := utils.GetSplitScreen()
	sysrootPath := filepath.Join(dir, "sysroot")
	if _, err := os.Stat(sysrootPath); os.IsNotExist(err) {
		sysrootPath = filepath.Join(dir, "dirs")
	}
	if _, err := os.Stat(sysrootPath); err == nil {
		if ss != nil {
			if suitName != "" {
				ss.SetAction("%s, applying system configuration (sysroot)...", suitName)
			} else {
				ss.SetAction("Applying system configuration (sysroot)...")
			}
		}
		if dryRun {
			logToFile(fmt.Sprintf("[DRY-RUN] Would rsync sysroot configuration %s -> /", sysrootPath))
			if !isAccessory && ss != nil {
				ss.AddStep(fmt.Sprintf("%s[OK]%s System configuration applied (simulated)", utils.ColorGreen, utils.ColorReset))
			}
		} else {
			cmd := fmt.Sprintf("rsync -aAX %s/ /", sysrootPath)
			err := utils.ExecTee(cmd, tailorLogFile)
			if !isAccessory && ss != nil {
				if err != nil {
					ss.AddStep(fmt.Sprintf("%s[WARN]%s System configuration applied with warnings", utils.ColorYellow, utils.ColorReset))
				} else {
					ss.AddStep(fmt.Sprintf("%s[OK]%s System configuration applied", utils.ColorGreen, utils.ColorReset))
				}
			}
		}
	}
}

// executeFinalizeCommands runs finalization scripts and commands
func executeFinalizeCommands(cmds []string, dir string, suitName string, dryRun bool) {
	if len(cmds) == 0 {
		return
	}
	ss := utils.GetSplitScreen()
	if ss != nil {
		if suitName != "" {
			ss.SetAction("%s, running finalization scripts (%d commands)...", suitName, len(cmds))
		} else {
			ss.SetAction("Running finalization scripts (%d commands)...", len(cmds))
		}
	}
	for idx, command := range cmds {
		fields := strings.Fields(command)
		cmdName := command
		if len(fields) > 0 {
			cmdName = filepath.Base(fields[0])
		}
		if ss != nil {
			if suitName != "" {
				ss.SetAction("%s, [%d/%d] Finalization: %s", suitName, idx+1, len(cmds), cmdName)
			} else {
				ss.SetAction("[%d/%d] Finalization: %s", idx+1, len(cmds), cmdName)
			}
		}
		if dryRun {
			logToFile(fmt.Sprintf("[DRY-RUN] Would run finalization command: %s", command))
			continue
		}
		if len(fields) > 0 {
			relScript := filepath.Join(dir, fields[0])
			if stat, err := os.Stat(relScript); err == nil && !stat.IsDir() {
				rest := strings.TrimSpace(command[len(fields[0]):])
				var fullCmd string
				if rest != "" {
					fullCmd = fmt.Sprintf("%s %s", relScript, rest)
				} else {
					fullCmd = fmt.Sprintf("%s %s", relScript, suitName)
				}
				_ = utils.ExecTee(fullCmd, tailorLogFile)
				continue
			}
		}
		_ = utils.ExecTee(command, tailorLogFile)
	}
	if ss != nil {
		suffix := ""
		if dryRun {
			suffix = " (simulated)"
		}
		ss.AddStep(fmt.Sprintf("%s[OK]%s Finalization completed%s", utils.ColorGreen, utils.ColorReset, suffix))
	}
}

func copySkelToUser(dryRun bool) {
	targetUser := os.Getenv("SUDO_USER")
	var userHome string
	if targetUser != "" {
		userHome = filepath.Join("/home", targetUser)
	} else if u := firstHumanUser(); u != nil {
		targetUser = u.Username
		userHome = u.HomeDir
	}

	if targetUser == "" || targetUser == "root" {
		logToFile("WARNING: unable to determine a non-root target user, skipping /etc/skel sync")
		return
	}

	if dryRun {
		logToFile(fmt.Sprintf("[DRY-RUN] Would sync /etc/skel -> %s", userHome))
		return
	}

	logToFile(fmt.Sprintf("Syncing /etc/skel -> %s", userHome))
	cmd := fmt.Sprintf("rsync -a --no-o --no-g --chown=%s:%s /etc/skel/ %s/", targetUser, targetUser, userHome)
	_ = utils.ExecTee(cmd, tailorLogFile)
}

func waitKeyPress(message string) {
	if !isInteractiveTerminal() {
		return
	}
	if message == "" {
		message = "Press Enter to continue..."
	}
	fmt.Printf("\n%s%s%s%s\n", utils.ColorCyan, utils.ColorBold, message, utils.ColorReset)
	reader := bufio.NewReader(os.Stdin)
	_, _ = reader.ReadString('\n')
}

func rebootSystem() {
	fmt.Printf("\n%s%s🔄 Costume requires a reboot. Restarting system in 3 seconds...%s\n", utils.ColorYellow, utils.ColorBold, utils.ColorReset)
	logToFile("Costume requires a reboot. Initiating system restart...")
	time.Sleep(3 * time.Second)
	if err := exec.Command("systemctl", "reboot").Run(); err != nil {
		if err2 := exec.Command("reboot").Run(); err2 != nil {
			_ = exec.Command("shutdown", "-r", "now").Run()
		}
	}
}
