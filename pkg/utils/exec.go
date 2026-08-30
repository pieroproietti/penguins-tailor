package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func ensureRootPath() {
	if os.Geteuid() != 0 {
		return
	}

	const rootPath = "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

	if os.Getenv("PATH") != rootPath {
		os.Setenv("PATH", rootPath)
	}
}

// Exec executes a shell command and displays real-time output in the terminal.
func Exec(command string) error {
	ensureRootPath()

	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ExecTee executes a shell command with interactive stdin, streaming stdout and stderr
// both to the terminal (or DECSTBM scrolling region) and to the specified log file.
func ExecTee(command string, logFilePath string) error {
	ensureRootPath()

	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
		// fallback
	}

	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return Exec(command)
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("[%s] EXEC: %s\n", timestamp, command))

	cmd := exec.Command("sh", "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = io.MultiWriter(os.Stdout, f)
	cmd.Stderr = io.MultiWriter(os.Stderr, f)
	return cmd.Run()
}

// ExecInteractive executes an interactive command. If split screen is active,
// it temporarily releases the scrolling region to allow fullscreen curses/dialog interfaces,
// restoring it upon completion.
func ExecInteractive(command string, logFilePath string) error {
	ensureRootPath()

	if ss := GetSplitScreen(); ss != nil && ss.IsActive() {
		return ss.ExecInteractive(command, logFilePath)
	}

	return ExecTee(command, logFilePath)
}

// ExecQuiet executes a command without printing anything to stdout/stderr.
func ExecQuiet(command string) error {
	ensureRootPath()

	cmd := exec.Command("sh", "-c", command)
	return cmd.Run()
}

// ExecLog executes a shell command and redirects stdout and stderr to the specified log file.
func ExecLog(command string, logFilePath string) error {
	return ExecLogMonitor(command, logFilePath, nil)
}

// ExecLogMonitor executes a shell command, writing everything to logfile and notifying onLine per line.
func ExecLogMonitor(command string, logFilePath string, onLine func(line string)) error {
	ensureRootPath()

	if err := os.MkdirAll(filepath.Dir(logFilePath), 0755); err != nil {
		// In case of directory error, continue
	}

	f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return ExecQuiet(command)
	}
	defer f.Close()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("[%s] EXEC: %s\n", timestamp, command))

	cmd := exec.Command("sh", "-c", command)

	r, w, err := os.Pipe()
	if err != nil {
		cmd.Stdout = f
		cmd.Stderr = f
		return cmd.Run()
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
	for scanner.Scan() {
		line := scanner.Text()
		f.WriteString(line + "\n")
		if onLine != nil {
			onLine(line)
		}
	}
	r.Close()

	return cmd.Wait()
}

// ExecCapture executes a command and returns the output as a string.
func ExecCapture(command string) (string, error) {
	ensureRootPath()

	var out bytes.Buffer
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = &out
	return out.String(), cmd.Run()
}

// ExecCaptureCombined executes a command and returns combined stdout and stderr as a string.
func ExecCaptureCombined(command string) (string, error) {
	ensureRootPath()

	var out bytes.Buffer
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}
