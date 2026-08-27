// Copyright 2026 Piero Proietti <piero.proietti@gmail.com>.
// All rights reserved.

package utils

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// Custom Bubble Tea message types
type actionMsg string
type stepMsg string
type logMsg string
type clearLogMsg struct{}

// tuiModel is the Bubble Tea Model for the split screen layout
type tuiModel struct {
	icon          string
	title         string
	subtitle      string
	completed     []string
	currentAction string
	startTime     time.Time
	spinner       spinner.Model
	viewport      viewport.Model
	logLines      []string
	maxLogLines   int
	width         int
	height        int
	ready         bool
	quitting      bool
}

func initialModel(icon, title, subtitle string, width, height int) tuiModel {
	s := spinner.New()
	s.Spinner = spinner.Spinner{
		Frames: []string{"|", "/", "-", "\\"},
		FPS:    100 * time.Millisecond,
	}
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)

	m := tuiModel{
		icon:        icon,
		title:       title,
		subtitle:    subtitle,
		completed:   make([]string, 0),
		startTime:   time.Now(),
		spinner:     s,
		logLines:    make([]string, 0),
		maxLogLines: 1000,
		width:       width,
		height:      height,
	}
	m.recalcLayout()
	return m
}

func (m tuiModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *tuiModel) recalcLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	headerHeight := 3
	if m.subtitle != "" {
		headerHeight = 4
	}

	maxSteps := 4
	if m.height >= 30 {
		maxSteps = 8
	} else if m.height >= 24 {
		maxSteps = 5
	}

	topHeight := headerHeight + maxSteps + 1 // +1 for current action line
	separatorHeight := 1
	vpHeight := m.height - topHeight - separatorHeight
	if vpHeight < 3 {
		vpHeight = 3
	}

	if !m.ready {
		m.viewport = viewport.New(m.width, vpHeight)
		m.viewport.SetContent(strings.Join(m.logLines, "\n"))
		m.viewport.GotoBottom()
		m.ready = true
	} else {
		m.viewport.Width = m.width
		m.viewport.Height = vpHeight
	}
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}
		var vpCmd tea.Cmd
		m.viewport, vpCmd = m.viewport.Update(msg)
		cmds = append(cmds, vpCmd)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()

	case spinner.TickMsg:
		var spinCmd tea.Cmd
		m.spinner, spinCmd = m.spinner.Update(msg)
		cmds = append(cmds, spinCmd)

	case actionMsg:
		m.currentAction = string(msg)

	case stepMsg:
		m.completed = append(m.completed, string(msg))
		m.currentAction = ""

	case logMsg:
		line := string(msg)
		if len(m.logLines) >= m.maxLogLines {
			m.logLines = m.logLines[len(m.logLines)-m.maxLogLines+1:]
		}
		m.logLines = append(m.logLines, line)
		if m.ready {
			m.viewport.SetContent(strings.Join(m.logLines, "\n"))
			m.viewport.GotoBottom()
		}

	case clearLogMsg:
		m.logLines = make([]string, 0)
		if m.ready {
			m.viewport.SetContent("")
		}
	}

	return m, tea.Batch(cmds...)
}

func (m tuiModel) View() string {
	if m.quitting || m.width <= 0 || m.height <= 0 {
		return ""
	}

	divWidth := m.width
	if divWidth > 72 {
		divWidth = 72
	}
	divider := strings.Repeat("═", divWidth)
	divStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	subStyle := lipgloss.NewStyle().Faint(true)

	var topLines []string
	topLines = append(topLines, divStyle.Render(divider))
	topLines = append(topLines, fmt.Sprintf("  %s %s", m.icon, titleStyle.Render(m.title)))
	if m.subtitle != "" {
		topLines = append(topLines, fmt.Sprintf("  %s", subStyle.Render(m.subtitle)))
	}
	topLines = append(topLines, divStyle.Render(divider))

	// Determine visible completed steps
	maxSteps := 4
	if m.height >= 30 {
		maxSteps = 8
	} else if m.height >= 24 {
		maxSteps = 5
	}

	var visible []string
	if len(m.completed) <= maxSteps {
		visible = m.completed
	} else {
		visible = m.completed[len(m.completed)-maxSteps:]
	}

	for i := 0; i < maxSteps; i++ {
		if i < len(visible) {
			topLines = append(topLines, "  "+visible[i])
		} else {
			topLines = append(topLines, "")
		}
	}

	// Current action / spinner line
	if m.currentAction != "" {
		elapsed := time.Since(m.startTime)
		mins := int(elapsed.Minutes())
		secs := int(elapsed.Seconds()) % 60
		timeStr := lipgloss.NewStyle().Faint(true).Render(fmt.Sprintf("[%02d:%02d]", mins, secs))
		actionText := m.currentAction
		topLines = append(topLines, fmt.Sprintf("  [%s] %s %s", m.spinner.View(), actionText, timeStr))
	} else {
		topLines = append(topLines, "")
	}

	topSection := strings.Join(topLines, "\n")

	// Separator line
	tag := " CONSOLE INTERATTIVA "
	totalLen := m.width
	if totalLen > 72 {
		totalLen = 72
	}
	leftLen := 4
	rightLen := totalLen - leftLen - len(tag)
	if rightLen < 4 {
		rightLen = 4
	}

	sepLine := fmt.Sprintf("%s%s%s",
		divStyle.Render(strings.Repeat("─", leftLen)),
		titleStyle.Render(tag),
		divStyle.Render(strings.Repeat("─", rightLen)),
	)

	// Bottom viewport
	vpSection := m.viewport.View()

	return lipgloss.JoinVertical(lipgloss.Left, topSection, sepLine, vpSection)
}

// SplitScreen controls the active Bubble Tea TUI session
type SplitScreen struct {
	mu      sync.Mutex
	program *tea.Program
	active  bool
	done    chan struct{}
}

var globalSplitScreen *SplitScreen
var splitMu sync.Mutex

// GetSplitScreen returns the active SplitScreen instance if any
func GetSplitScreen() *SplitScreen {
	splitMu.Lock()
	defer splitMu.Unlock()
	return globalSplitScreen
}

// StartSplitScreen initializes the horizontal split screen TUI on terminal
func StartSplitScreen(icon, title, subtitle string) *SplitScreen {
	splitMu.Lock()
	defer splitMu.Unlock()

	if !isTerminal() {
		return nil
	}

	rows, cols := GetTerminalSize()
	if rows < 14 {
		return nil
	}

	model := initialModel(icon, title, subtitle, cols, rows)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithoutCatchPanics())

	ss := &SplitScreen{
		program: p,
		active:  true,
		done:    make(chan struct{}),
	}

	go func() {
		_, _ = p.Run()
		close(ss.done)
	}()

	globalSplitScreen = ss
	return ss
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

// AddStep appends a completed step to the top pane
func (ss *SplitScreen) AddStep(step string) {
	if ss == nil {
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if ss.active && ss.program != nil {
		ss.program.Send(stepMsg(step))
	}
}

// SetAction sets the current action description shown on the animated spinner line
func (ss *SplitScreen) SetAction(format string, a ...interface{}) {
	if ss == nil {
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if ss.active && ss.program != nil {
		ss.program.Send(actionMsg(fmt.Sprintf(format, a...)))
	}
}

// Log sends a line of output to the bottom interactive console
func (ss *SplitScreen) Log(line string) {
	if ss == nil {
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	if ss.active && ss.program != nil {
		ss.program.Send(logMsg(line))
	}
}

// ExecStream runs a shell command and streams stdout and stderr line-by-line
// to the bottom interactive console and to the log file.
func (ss *SplitScreen) ExecStream(command string, logFilePath string) error {
	ensureRootPath()

	var f *os.File
	if logFilePath != "" {
		if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err == nil {
			if file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				f = file
				defer f.Close()
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				f.WriteString(fmt.Sprintf("[%s] EXEC: %s\n", timestamp, command))
			}
		}
	}

	cmd := exec.Command("sh", "-c", command)
	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	cmd.Stdout = w
	cmd.Stderr = w

	if err := cmd.Start(); err != nil {
		w.Close()
		r.Close()
		return err
	}
	w.Close()

	scanner := bufio.NewScanner(r)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if f != nil {
			f.WriteString(line + "\n")
		}
		ss.Log(line)
	}
	r.Close()

	return cmd.Wait()
}

// ExecInteractive temporarily suspends the TUI, hands over full terminal I/O
// to the interactive command (e.g. Debconf dialog/readline prompts), and
// restores the TUI cleanly upon completion.
func (ss *SplitScreen) ExecInteractive(command string, logFilePath string) error {
	ensureRootPath()

	if logFilePath != "" {
		if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err == nil {
			if f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
				timestamp := time.Now().Format("2006-01-02 15:04:05")
				f.WriteString(fmt.Sprintf("[%s] EXEC (INTERACTIVE): %s\n", timestamp, command))
				f.Close()
			}
		}
	}

	ss.mu.Lock()
	if ss.program != nil {
		_ = ss.program.ReleaseTerminal()
	}
	ss.mu.Unlock()

	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	ss.mu.Lock()
	if ss.program != nil {
		_ = ss.program.RestoreTerminal()
		rows, cols := GetTerminalSize()
		ss.program.Send(tea.WindowSizeMsg{Width: cols, Height: rows})
	}
	ss.mu.Unlock()

	return err
}

// Close finishes the split screen TUI and restores standard terminal state
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
	if ss.program != nil {
		ss.program.Quit()
	}
	ss.mu.Unlock()

	// Wait for program to exit cleanly
	select {
	case <-ss.done:
	case <-time.After(500 * time.Millisecond):
		if ss.program != nil {
			ss.program.Kill()
		}
	}

	splitMu.Lock()
	globalSplitScreen = nil
	splitMu.Unlock()
}
