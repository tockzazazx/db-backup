// Package schedule manages the systemd timer that runs "boxdb upload"
// automatically. Only one schedule is supported: a daily run at a fixed time.
package schedule

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	unitDir     = "/etc/systemd/system"
	serviceFile = "boxdb-upload.service"
	timerFile   = "boxdb-upload.timer"
)

// Daily describes the schedule: one upload per day.
type Daily struct {
	Time string // "HH:MM", 24-hour clock
	User string // system user the upload runs as (owns the boxdb config)
	Exec string // absolute path to the boxdb binary
}

// ParseTime validates and normalizes a 24-hour "HH:MM" string.
func ParseTime(s string) (string, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return "", fmt.Errorf("invalid time %q — use 24-hour HH:MM, e.g. --daily 03:00", s)
	}
	return t.Format("15:04"), nil
}

func serviceUnit(d Daily) string {
	return fmt.Sprintf(`[Unit]
Description=boxdb upload to S3
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
User=%s
ExecStart=%s upload
`, d.User, d.Exec)
}

func timerUnit(d Daily) string {
	return fmt.Sprintf(`[Unit]
Description=Daily boxdb upload at %s

[Timer]
OnCalendar=*-*-* %s:00
Persistent=true

[Install]
WantedBy=timers.target
`, d.Time, d.Time)
}

func requireSystemd() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("schedule requires Linux with systemd (this is %s)", runtime.GOOS)
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return fmt.Errorf("systemctl not found — schedule requires systemd")
	}
	return nil
}

// Installed reports whether the timer unit file exists.
func Installed() bool {
	_, err := os.Stat(filepath.Join(unitDir, timerFile))
	return err == nil
}

// Install writes the systemd units and enables the timer. Needs root.
func Install(d Daily) error {
	if err := requireSystemd(); err != nil {
		return err
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("writing to %s needs root — run: sudo boxdb schedule --daily %s", unitDir, d.Time)
	}
	if err := os.WriteFile(filepath.Join(unitDir, serviceFile), []byte(serviceUnit(d)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(unitDir, timerFile), []byte(timerUnit(d)), 0o644); err != nil {
		return err
	}
	if err := systemctl("daemon-reload"); err != nil {
		return err
	}
	return systemctl("enable", "--now", timerFile)
}

// Remove disables the timer and deletes the unit files. Needs root.
func Remove() error {
	if err := requireSystemd(); err != nil {
		return err
	}
	if !Installed() {
		return fmt.Errorf("no schedule installed")
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("removing the schedule needs root — run: sudo boxdb schedule --remove")
	}
	if err := systemctl("disable", "--now", timerFile); err != nil {
		return err
	}
	os.Remove(filepath.Join(unitDir, serviceFile))
	os.Remove(filepath.Join(unitDir, timerFile))
	return systemctl("daemon-reload")
}

// Info describes the installed schedule for display.
type Info struct {
	Time       string // configured HH:MM
	User       string
	Exec       string // full ExecStart line
	Active     string // systemd active state of the timer
	NextRun    string
	LastRun    string
	LastResult string
}

// Status reads the installed schedule. Returns nil when none is installed.
func Status() (*Info, error) {
	if err := requireSystemd(); err != nil {
		return nil, err
	}
	if !Installed() {
		return nil, nil
	}

	info := &Info{}
	if data, err := os.ReadFile(filepath.Join(unitDir, serviceFile)); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if v, ok := strings.CutPrefix(line, "User="); ok {
				info.User = v
			}
			if v, ok := strings.CutPrefix(line, "ExecStart="); ok {
				info.Exec = v
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(unitDir, timerFile)); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if v, ok := strings.CutPrefix(line, "OnCalendar=*-*-* "); ok {
				info.Time = strings.TrimSuffix(v, ":00")
			}
		}
	}
	info.Active = systemctlOut("is-active", timerFile)
	info.NextRun = systemctlOut("show", timerFile, "--property=NextElapseUSecRealtime", "--value")
	info.LastRun = systemctlOut("show", timerFile, "--property=LastTriggerUSec", "--value")
	info.LastResult = systemctlOut("show", serviceFile, "--property=Result", "--value")
	return info, nil
}

func systemctl(args ...string) error {
	out, err := exec.Command("systemctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func systemctlOut(args ...string) string {
	out, _ := exec.Command("systemctl", args...).Output()
	return strings.TrimSpace(string(out))
}
