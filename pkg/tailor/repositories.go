package tailor

import (
	"os"
	"regexp"
	"strings"

	"github.com/pieroproietti/penguins-tailor/pkg/utils"
)

// setupRepositories applies the "repositories" section of the nested suit format
func setupRepositories(repos *Repositories, suitName string, dryRun bool) {
	if repos == nil {
		return
	}

	if len(repos.SourcesList) > 0 {
		if dryRun {
			logToFile(WarnPrefix(suitName) + "[DRY-RUN] Would enable apt components: " + strings.Join(repos.SourcesList, " "))
		} else {
			if err := enableAptComponents(repos.SourcesList); err != nil {
				logToFile(WarnPrefix(suitName) + "sources.list: " + err.Error())
			}
		}
	}

	if len(repos.SourcesListD) > 0 {
		logToFile(WarnPrefix(suitName) + "running third-party repository setup commands...")
		for idx, command := range repos.SourcesListD {
			if ss := utils.GetSplitScreen(); ss != nil {
				ss.SetAction("Third-party repo [%d/%d]", idx+1, len(repos.SourcesListD))
			}
			if dryRun {
				logToFile(WarnPrefix(suitName) + "[DRY-RUN] Would execute repository command: " + command)
				continue
			}
			if err := utils.ExecTee(command, tailorLogFile); err != nil {
				logToFile(WarnPrefix(suitName) + "repository command failed: " + command + ": " + err.Error())
			}
		}
	}

	if repos.Update {
		logToFile(WarnPrefix(suitName) + "apt-get update...")
		if ss := utils.GetSplitScreen(); ss != nil {
			ss.SetAction("Updating package index (apt-get update)...")
		}
		if !dryRun {
			_ = utils.ExecTee("apt-get update", tailorLogFile)
		}
	}

	if repos.Upgrade {
		logToFile(WarnPrefix(suitName) + "apt-get upgrade...")
		if ss := utils.GetSplitScreen(); ss != nil {
			ss.SetAction("Upgrading packages (apt-get upgrade)...")
		}
		if !dryRun {
			_ = utils.ExecTee("UCF_FORCE_CONFFOLD=1 DEBIAN_FRONTEND=readline apt-get -o Dpkg::Options::='--force-confdef' -o Dpkg::Options::='--force-confold' upgrade -y", tailorLogFile)
		}
	}
}

// WarnPrefix generates a consistent prefix for logs of this package.
func WarnPrefix(suitName string) string {
	return "[" + suitName + "] "
}

// enableAptComponents ensures the requested components are enabled
func enableAptComponents(components []string) error {
	const path = "/etc/apt/sources.list"

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	debLineRe := regexp.MustCompile(`^(deb|deb-src)\s+(\S+)\s+(\S+)\s+(.*)$`)

	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		m := debLineRe.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		existing := strings.Fields(m[4])
		existingSet := make(map[string]struct{}, len(existing))
		for _, c := range existing {
			existingSet[c] = struct{}{}
		}

		added := false
		for _, want := range components {
			if _, ok := existingSet[want]; !ok {
				existing = append(existing, want)
				existingSet[want] = struct{}{}
				added = true
			}
		}

		if added {
			lines[i] = strings.Join([]string{m[1], m[2], m[3], strings.Join(existing, " ")}, " ")
			changed = true
		}
	}

	if !changed {
		return nil
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}
