// Copyright 2026 Piero Proietti <piero.proietti@gmail.com>.
// All rights reserved.

package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

// GetTerminalSize returns terminal rows and columns, or (24, 80) as default fallback.
func GetTerminalSize() (int, int) {
	ws := &winsize{}
	ret, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))
	if int(ret) == 0 && ws.Row > 0 && ws.Col > 0 {
		return int(ws.Row), int(ws.Col)
	}
	return 24, 80
}

// SplitScreenConfig defines the structured header information for the split screen top pane.
type SplitScreenConfig struct {
	Icon    string
	Atelier string
	System  string
	Costume string
	Branch  string
	Notes   string
}

// FormatHeaderLines constructs the header lines according to layout requirements:
// Line 1: Atelier: <remote/origin> (with branch if not default)
// Line 2: Distribution: <distribution>-<codename>
// Line 3: Costume: <costume> (or Accessory: <accessory>)
// Line 4: <description>
func FormatHeaderLines(cfg SplitScreenConfig) []string {
	var lines []string

	icon := cfg.Icon

	// Line 1: Atelier
	if cfg.Atelier != "" {
		atelierVal := cfg.Atelier
		if cfg.Branch != "" && cfg.Branch != "main" && cfg.Branch != "master" {
			atelierVal = fmt.Sprintf("%s (%s)", cfg.Atelier, cfg.Branch)
		}
		var line1 string
		if icon != "" {
			line1 = fmt.Sprintf("  %s %sAtelier:%s %s", icon, colorize(ColorBold+ColorWhite), colorize(ColorReset), atelierVal)
		} else {
			line1 = fmt.Sprintf("  %sAtelier:%s %s", colorize(ColorBold+ColorWhite), colorize(ColorReset), atelierVal)
		}
		lines = append(lines, line1)
	}

	if cfg.System != "" {
		indent := "  "
		if icon != "" {
			indent = "     "
		}
		lines = append(lines, fmt.Sprintf("%s%sDistribution:%s %s", indent, colorize(ColorBold+ColorWhite), colorize(ColorReset), cfg.System))
	}

	// Costume / Accessory
	if cfg.Costume != "" {
		var line2 string
		indent := "  "
		if icon != "" {
			indent = "     "
			if cfg.Atelier == "" {
				indent = fmt.Sprintf("  %s ", icon)
			}
		}
		if strings.Contains(cfg.Costume, ":") {
			parts := strings.SplitN(cfg.Costume, ":", 2)
			label := strings.TrimSpace(parts[0]) + ":"
			val := strings.TrimSpace(parts[1])
			if val != "" {
				line2 = fmt.Sprintf("%s%s%s%s %s", indent, colorize(ColorBold+ColorWhite), label, colorize(ColorReset), val)
			} else {
				line2 = fmt.Sprintf("%s%s%s%s", indent, colorize(ColorBold+ColorWhite), label, colorize(ColorReset))
			}
		} else {
			line2 = fmt.Sprintf("%s%sCostume:%s %s", indent, colorize(ColorBold+ColorWhite), colorize(ColorReset), cfg.Costume)
		}
		lines = append(lines, line2)
	}

	// Line 3: Description (without label prefix)
	if cfg.Notes != "" {
		indent := "  "
		if icon != "" {
			indent = "     "
		}
		line3 := fmt.Sprintf("%s%s%s%s", indent, colorize(ColorDim), cfg.Notes, colorize(ColorReset))
		lines = append(lines, line3)
	}

	if len(lines) == 0 {
		if icon != "" {
			lines = append(lines, fmt.Sprintf("  %s", icon))
		} else {
			lines = append(lines, "  Costume")
		}
	}

	return lines
}

type SplitScreen struct {
	mu            sync.Mutex
	active        bool
	totalCols     int
	headerLines   []string
	currentAction string
	currentOpen   bool
	currentSince  time.Time
	installed     int
	unavailable   int
	failed        int
	interactive   bool
	stopChan      chan struct{}
}

var globalSplitScreen *SplitScreen
var splitMu sync.Mutex

// GetSplitScreen returns the active SplitScreen instance if any
func GetSplitScreen() *SplitScreen {
	splitMu.Lock()
	defer splitMu.Unlock()
	return globalSplitScreen
}

// StartSplitScreenConfig initializes the progressive terminal renderer.
func StartSplitScreenConfig(cfg SplitScreenConfig) *SplitScreen {
	splitMu.Lock()
	defer splitMu.Unlock()

	if !isTerminal() {
		return nil
	}

	_, cols := GetTerminalSize()

	headerLines := FormatHeaderLines(cfg)

	ss := &SplitScreen{
		active:      true,
		totalCols:   cols,
		headerLines: headerLines,
		stopChan:    make(chan struct{}),
	}

	ss.drawHeader()

	// Start background spinner updater for the status line
	go ss.spinnerLoop()

	globalSplitScreen = ss
	return ss
}

// StartSplitScreen provides backwards compatibility with icon, title, subtitle
func StartSplitScreen(icon, title, subtitle string) *SplitScreen {
	return StartSplitScreenConfig(SplitScreenConfig{
		Icon:    icon,
		Costume: title,
		Notes:   subtitle,
	})
}

func (ss *SplitScreen) drawHeader() {
	divWidth := ss.totalCols
	if divWidth > 72 {
		divWidth = 72
	}
	divider := strings.Repeat("═", divWidth)

	for _, line := range ss.headerLines {
		fmt.Printf("%s\n", line)
	}
	fmt.Printf("%s%s%s\n", colorize(ColorCyan), divider, colorize(ColorReset))
}

func (ss *SplitScreen) drawInteractiveHeader() {
	tag := " INTERACTIVE CONSOLE "
	totalLen := ss.totalCols
	if totalLen > 72 {
		totalLen = 72
	}
	leftLen := 4
	rightLen := totalLen - leftLen - len(tag)
	if rightLen < 4 {
		rightLen = 4
	}

	sepLine := fmt.Sprintf("%s%s%s%s%s%s%s",
		colorize(ColorCyan),
		strings.Repeat("─", leftLen),
		colorize(ColorBold+ColorWhite),
		tag,
		colorize(ColorReset+ColorCyan),
		strings.Repeat("─", rightLen),
		colorize(ColorReset),
	)

	fmt.Printf("%s\n", sepLine)
}

func (ss *SplitScreen) spinnerLoop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	idx := 0

	for {
		select {
		case <-ss.stopChan:
			return
		case <-ticker.C:
			ss.mu.Lock()
			if !ss.active {
				ss.mu.Unlock()
				return
			}
			if ss.interactive {
				ss.mu.Unlock()
				continue
			}
			if !ss.currentOpen {
				ss.mu.Unlock()
				continue
			}

			frame := asciiSpinnerFrames[idx%len(asciiSpinnerFrames)]
			idx++
			ss.renderCurrentLocked(frame)
			ss.mu.Unlock()
		}
	}
}

// IsActive reports whether the TUI split screen is currently running
func (ss *SplitScreen) IsActive() bool {
	if ss == nil {
		return false
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.active
}

// AddStep finalizes the current action when one exists, otherwise it writes a
// standalone historical step.
func (ss *SplitScreen) AddStep(step string) {
	if ss == nil {
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.finishCurrentLocked(step)
}

// SetAction starts a new mutable terminal line. A prior current line is first
// finalized so no dynamic output can be overwritten accidentally.
func (ss *SplitScreen) SetAction(format string, a ...interface{}) {
	if ss == nil {
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()

	if ss.currentOpen {
		ss.finishCurrentLocked(ss.neutralCurrentLine())
	}
	ss.currentAction = fmt.Sprintf(format, a...)
	ss.currentSince = time.Now()
	ss.currentOpen = true
	ss.renderCurrentLocked(asciiSpinnerFrames[0])
}

// SetPackageSummary updates the global package outcome counters shown by the
// dashboard. Rendering stays inside SplitScreen so callers provide data only.
func (ss *SplitScreen) SetPackageSummary(installed, unavailable, failed int) {
	if ss == nil {
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.installed = installed
	ss.unavailable = unavailable
	ss.failed = failed
}

// AddPackageStep records a concise package outcome for a costume or accessory.
func (ss *SplitScreen) AddPackageStep(subject string, installed, unavailable, failed int) {
	if ss == nil {
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()

	color := ColorCyan
	if failed > 0 {
		color = ColorYellow
	}
	ss.finishCurrentLocked(fmt.Sprintf("%s--> %s — %s%s", colorize(color), subject, packageSummaryText(installed, unavailable, failed), colorize(ColorReset)))
}

func (ss *SplitScreen) renderCurrentLocked(frame string) {
	fmt.Printf("\r\033[2K%s", ss.actionLine(ss.currentAction, frame, time.Since(ss.currentSince)))
}

func (ss *SplitScreen) finishCurrentLocked(line string) {
	if ss.currentOpen {
		fmt.Printf("\r\033[2K%s\n", line)
		ss.currentAction = ""
		ss.currentOpen = false
		return
	}
	fmt.Printf("%s\n", line)
}

func (ss *SplitScreen) neutralCurrentLine() string {
	return fmt.Sprintf("%s[INFO]%s %s", colorize(ColorCyan), colorize(ColorReset), ss.currentAction)
}

func (ss *SplitScreen) actionLine(action, frame string, elapsed time.Duration) string {
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60
	return fmt.Sprintf("  %s[%s%s%s]%s %s %s[%02d:%02d]%s",
		colorize(ColorCyan),
		colorize(ColorBold+ColorWhite), frame, colorize(ColorReset+ColorCyan),
		colorize(ColorReset),
		action,
		colorize(ColorDim), mins, secs, colorize(ColorReset))
}

func packageSummaryText(installed, unavailable, failed int) string {
	return fmt.Sprintf("%d installed · %d unavailable · %d failed", installed, unavailable, failed)
}

// ExecInteractive finalizes the mutable line, gives the command direct
// terminal control, then resumes the progressive output below the command.
func (ss *SplitScreen) ExecInteractive(command string, logFilePath string) error {
	ensureRootPath()

	ss.mu.Lock()
	ss.interactive = true
	if ss.currentOpen {
		ss.finishCurrentLocked(ss.neutralCurrentLine())
	}
	ss.drawInteractiveHeader()
	ss.mu.Unlock()

	var f *os.File
	if logFilePath != "" {
		if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err == nil {
			if file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				f = file
				defer f.Close()
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				f.WriteString(fmt.Sprintf("[%s] EXEC (INTERACTIVE): %s\n", timestamp, command))
			}
		}
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = os.Stdin
	if f != nil {
		cmd.Stdout = io.MultiWriter(os.Stdout, f)
		cmd.Stderr = io.MultiWriter(os.Stderr, f)
	} else {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()

	ss.mu.Lock()
	ss.interactive = false
	ss.mu.Unlock()

	return err
}

// Close finalizes progressive output and leaves the cursor on the next line.
func (ss *SplitScreen) Close() {
	if ss == nil {
		return
	}
	ss.mu.Lock()
	if !ss.active {
		ss.mu.Unlock()
		return
	}
	if ss.currentOpen {
		ss.finishCurrentLocked(ss.neutralCurrentLine())
	}
	fmt.Printf("Packages: %s\n", packageSummaryText(ss.installed, ss.unavailable, ss.failed))
	ss.active = false
	close(ss.stopChan)
	ss.mu.Unlock()

	splitMu.Lock()
	globalSplitScreen = nil
	splitMu.Unlock()

	fmt.Print("\033[?25h")
}
