// Copyright 2026 Piero Proietti <piero.proietti@gmail.com>.
// All rights reserved.

package utils

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type Spinner struct {
	mu        sync.Mutex
	text      string
	subtext   string
	frames    []string
	interval  time.Duration
	stopChan  chan struct{}
	running   bool
	isTTY     bool
	startTime time.Time
}

// Robust ASCII frames compatible with any console (Linux TTY, VGA, SSH, serial)
var asciiSpinnerFrames = []string{"|", "/", "-", "\\"}

// isTerminal checks if stdout is an interactive terminal (TTY)
func isTerminal() bool {
	if DisableColors {
		return false
	}
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// NewSpinner creates a new spinner with the specified text
func NewSpinner(text string) *Spinner {
	return &Spinner{
		text:      text,
		frames:    asciiSpinnerFrames,
		interval:  100 * time.Millisecond,
		stopChan:  make(chan struct{}),
		isTTY:     isTerminal(),
		startTime: time.Now(),
	}
}

// Start begins the spinner animation with elapsed time indicator
func (s *Spinner) Start() *Spinner {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return s
	}
	s.running = true
	s.startTime = time.Now()
	s.stopChan = make(chan struct{})
	s.mu.Unlock()

	if !s.isTTY {
		fmt.Printf("  ... %s\n", s.text)
		return s
	}

	// Hide cursor during animation
	fmt.Print("\033[?25l")

	go func() {
		idx := 0
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopChan:
				return
			case <-ticker.C:
				s.mu.Lock()
				if !s.running {
					s.mu.Unlock()
					return
				}
				frame := s.frames[idx%len(s.frames)]
				text := s.text
				sub := s.subtext
				elapsed := time.Since(s.startTime)
				idx++
				s.mu.Unlock()

				mins := int(elapsed.Minutes())
				secs := int(elapsed.Seconds()) % 60
				timeStr := fmt.Sprintf("[%02d:%02d]", mins, secs)

				line := fmt.Sprintf("  %s[%s%s%s]%s %s %s%s%s",
					colorize(ColorCyan),
					colorize(ColorBold+ColorWhite), frame, colorize(ColorReset+ColorCyan),
					colorize(ColorReset),
					text,
					colorize(ColorDim), timeStr, colorize(ColorReset))

				if sub != "" {
					if len(sub) > 42 {
						sub = sub[:39] + "..."
					}
					line += fmt.Sprintf(" %s- %s%s", colorize(ColorDim), sub, colorize(ColorReset))
				}

				// Overwrite current line
				fmt.Printf("\r\033[2K%s", line)
			}
		}
	}()

	return s
}

// UpdateText updates the main message displayed by the spinner
func (s *Spinner) UpdateText(format string, a ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.text = fmt.Sprintf(format, a...)
}

// UpdateSubtext updates the detail sub-message (e.g. current package name)
func (s *Spinner) UpdateSubtext(format string, a ...interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.subtext = fmt.Sprintf(format, a...)
}

func (s *Spinner) stop() time.Duration {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return 0
	}
	s.running = false
	close(s.stopChan)
	elapsed := time.Since(s.startTime)
	s.mu.Unlock()

	if s.isTTY {
		// Restore cursor
		fmt.Print("\033[?25h")
	}
	return elapsed
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return ""
	}
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	if mins > 0 {
		return fmt.Sprintf(" (%dm %02ds)", mins, secs)
	}
	return fmt.Sprintf(" (%ds)", secs)
}

// Success stops the spinner and prints [OK] with duration
func (s *Spinner) Success(format string, a ...interface{}) {
	d := s.stop()
	msg := fmt.Sprintf(format, a...) + formatDuration(d)
	if s.isTTY {
		fmt.Printf("\r\033[2K  %s[OK]%s %s\n", colorize(ColorBold+ColorGreen), colorize(ColorReset), msg)
	} else {
		fmt.Printf("  [OK] %s\n", msg)
	}
}

// Fail stops the spinner and prints [FAIL] with duration
func (s *Spinner) Fail(format string, a ...interface{}) {
	d := s.stop()
	msg := fmt.Sprintf(format, a...) + formatDuration(d)
	if s.isTTY {
		fmt.Printf("\r\033[2K  %s[FAIL]%s %s\n", colorize(ColorBold+ColorRed), colorize(ColorReset), msg)
	} else {
		fmt.Printf("  [FAIL] %s\n", msg)
	}
}

// Warn stops the spinner and prints [WARN] with duration
func (s *Spinner) Warn(format string, a ...interface{}) {
	d := s.stop()
	msg := fmt.Sprintf(format, a...) + formatDuration(d)
	if s.isTTY {
		fmt.Printf("\r\033[2K  %s[WARN]%s %s\n", colorize(ColorBold+ColorYellow), colorize(ColorReset), msg)
	} else {
		fmt.Printf("  [WARN] %s\n", msg)
	}
}

// Info stops the spinner and prints [INFO]
func (s *Spinner) Info(format string, a ...interface{}) {
	s.stop()
	msg := fmt.Sprintf(format, a...)
	if s.isTTY {
		fmt.Printf("\r\033[2K  %s[INFO]%s %s\n", colorize(ColorBold+ColorCyan), colorize(ColorReset), msg)
	} else {
		fmt.Printf("  [INFO] %s\n", msg)
	}
}
