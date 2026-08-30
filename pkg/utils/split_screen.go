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
	Costume string
	Branch  string
	Notes   string
}

// FormatHeaderLines constructs the header lines according to layout requirements:
// 1ª riga: Atelier: <remote/origin> (con eventuale branch se non default)
// 2ª riga: Costume: <costume> (o Accessory: <accessory>)
// 3ª riga: <descrizione>
func FormatHeaderLines(cfg SplitScreenConfig) []string {
	var lines []string

	icon := cfg.Icon

	// 1ª riga: Atelier
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

	// 2ª riga: Costume / Accessory
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

	// 3ª riga: Descrizione (senza label prefisso)
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
	totalRows     int
	totalCols     int
	headerLines   []string
	headerRows    int
	maxSteps      int
	topHeight     int // row number of current action / spinner line
	sepRow        int // row number of separator line
	scrollStart   int // first row of bottom scrolling region
	completed     []string
	currentAction string
	startTime     time.Time
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

// StartSplitScreenConfig initializes the native DECSTBM split screen with structured config
func StartSplitScreenConfig(cfg SplitScreenConfig) *SplitScreen {
	splitMu.Lock()
	defer splitMu.Unlock()

	if !isTerminal() {
		return nil
	}

	rows, cols := GetTerminalSize()
	if rows < 14 {
		// Terminal too small for meaningful split screen
		return nil
	}

	headerLines := FormatHeaderLines(cfg)
	headerRows := len(headerLines) + 2 // +2 for top and bottom divider lines

	// Allarghiamo di un paio di righe la finestra dei passaggi completati
	maxSteps := 6
	if rows >= 40 {
		maxSteps = 14
	} else if rows >= 32 {
		maxSteps = 10
	} else if rows >= 24 {
		maxSteps = 7
	}

	topHeight := headerRows + maxSteps + 1
	sepRow := topHeight + 1
	scrollStart := sepRow + 1

	// Ensure there are at least 4 lines for bottom interactive console
	if scrollStart >= rows-3 {
		maxSteps = rows - headerRows - 5
		if maxSteps < 3 {
			maxSteps = 3
		}
		topHeight = headerRows + maxSteps + 1
		sepRow = topHeight + 1
		scrollStart = sepRow + 1
	}

	ss := &SplitScreen{
		active:        true,
		totalRows:     rows,
		totalCols:     cols,
		headerLines:   headerLines,
		headerRows:    headerRows,
		maxSteps:      maxSteps,
		topHeight:     topHeight,
		sepRow:        sepRow,
		scrollStart:   scrollStart,
		completed:     make([]string, 0),
		currentAction: "",
		startTime:     time.Now(),
		stopChan:      make(chan struct{}),
	}

	// Initial drawing of split screen layout
	ss.drawFullLayout()

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

	fmt.Printf("%s%s%s\n", colorize(ColorCyan), divider, colorize(ColorReset))
	for _, line := range ss.headerLines {
		fmt.Printf("%s\n", line)
	}
	fmt.Printf("%s%s%s\n", colorize(ColorCyan), divider, colorize(ColorReset))
}

func (ss *SplitScreen) drawFullLayout() {
	// Clear entire terminal and move to (1,1)
	fmt.Print("\033[2J\033[1;1H")

	// Draw top header
	ss.drawHeader()

	// Draw separator line at sepRow
	ss.drawSeparator(ss.sepRow, ss.totalCols)

	// Set DECSTBM scrolling region for the bottom area
	fmt.Printf("\033[%d;%dr", ss.scrollStart, ss.totalRows)

	// Position cursor in the bottom scrolling region
	fmt.Printf("\033[%d;1H", ss.scrollStart)
}

func (ss *SplitScreen) drawSeparator(row, cols int) {
	tag := " CONSOLE INTERATTIVA "
	totalLen := cols
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

	// Save cursor, draw separator at row, restore cursor
	fmt.Printf("\0337\033[%d;1H\033[2K%s\0338", row, sepLine)
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
			action := ss.currentAction
			if action == "" {
				ss.mu.Unlock()
				continue
			}

			frame := asciiSpinnerFrames[idx%len(asciiSpinnerFrames)]
			idx++
			elapsed := time.Since(ss.startTime)
			row := ss.topHeight
			ss.mu.Unlock()

			mins := int(elapsed.Minutes())
			secs := int(elapsed.Seconds()) % 60
			timeStr := fmt.Sprintf("[%02d:%02d]", mins, secs)

			line := fmt.Sprintf("  %s[%s%s%s]%s %s %s%s%s",
				colorize(ColorCyan),
				colorize(ColorBold+ColorWhite), frame, colorize(ColorReset+ColorCyan),
				colorize(ColorReset),
				action,
				colorize(ColorDim), timeStr, colorize(ColorReset))

			// DEC Save cursor, move to spinner row, clear line, write text, DEC Restore cursor
			fmt.Printf("\0337\033[%d;1H\033[2K%s\0338", row, line)
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

// AddStep appends a completed step to the top pane and refreshes the status view
func (ss *SplitScreen) AddStep(step string) {
	if ss == nil {
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.completed = append(ss.completed, step)
	ss.currentAction = ""
	ss.redrawStatusLocked()
}

// SetAction sets the current action description shown on the animated spinner line
func (ss *SplitScreen) SetAction(format string, a ...interface{}) {
	if ss == nil {
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.currentAction = fmt.Sprintf(format, a...)
}

func (ss *SplitScreen) redrawStatusLocked() {
	startRow := ss.headerRows + 1
	maxVisible := ss.maxSteps

	// Determine which completed steps to display
	var visible []string
	if len(ss.completed) <= maxVisible {
		visible = ss.completed
	} else {
		visible = ss.completed[len(ss.completed)-maxVisible:]
	}

	// Buffer all redraw operations
	var sb strings.Builder
	sb.WriteString("\0337") // save cursor

	for i := 0; i < maxVisible; i++ {
		row := startRow + i
		sb.WriteString(fmt.Sprintf("\033[%d;1H\033[2K", row))
		if i < len(visible) {
			sb.WriteString("  " + visible[i])
		}
	}

	// Clear current action row
	sb.WriteString(fmt.Sprintf("\033[%d;1H\033[2K", ss.topHeight))
	sb.WriteString("\0338") // restore cursor

	fmt.Print(sb.String())
}

// ExecInteractive temporarily suspends the split screen margins for a full-screen
// interactive command, then restores the split screen layout upon completion.
func (ss *SplitScreen) ExecInteractive(command string, logFilePath string) error {
	ensureRootPath()

	ss.mu.Lock()
	// Reset scroll region to full screen
	fmt.Print("\033[r\033[2J\033[1;1H")
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
	if ss.active {
		ss.drawFullLayout()
		ss.redrawStatusLocked()
	}
	ss.mu.Unlock()

	return err
}

// Close finishes the split screen, restores scrolling margins and moves cursor to bottom
func (ss *SplitScreen) Close() {
	if ss == nil {
		return
	}
	ss.mu.Lock()
	if !ss.active {
		ss.mu.Unlock()
		return
	}
	ss.active = false
	close(ss.stopChan)
	ss.mu.Unlock()

	splitMu.Lock()
	globalSplitScreen = nil
	splitMu.Unlock()

	// Reset DECSTBM scrolling region to full screen
	fmt.Print("\033[r")
	// Move cursor to bottom row and output newline
	fmt.Printf("\033[%d;1H\n", ss.totalRows)
	// Show cursor
	fmt.Print("\033[?25h")
}
