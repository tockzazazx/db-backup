// Package schedule manages the systemd timer that runs "boxdb upload"
// automatically. One schedule per machine: daily, weekly, or monthly.
package schedule

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	unitDir     = "/etc/systemd/system"
	serviceFile = "boxdb-upload.service"
	timerFile   = "boxdb-upload.timer"

	// descPrefix marks our units; Status reads the human description back
	// from the timer file instead of re-parsing the OnCalendar expression.
	descPrefix = "boxdb upload "
)

// Spec is one fully-described schedule ready to install.
type Spec struct {
	OnCalendar  string // systemd calendar expression
	Description string // human readable, e.g. "weekly on Saturday at 03:00"
	User        string // system user the upload runs as (owns the boxdb config)
	Exec        string // absolute path to the boxdb binary
}

// ParseTime validates and normalizes a 24-hour "HH:MM" string.
func ParseTime(s string) (string, error) {
	t, err := time.Parse("15:04", s)
	if err != nil {
		return "", fmt.Errorf("invalid time %q — use 24-hour HH:MM, e.g. 03:00", s)
	}
	return t.Format("15:04"), nil
}

// Daily builds a schedule that runs every day at the given time.
func Daily(at string) (Spec, error) {
	t, err := ParseTime(at)
	if err != nil {
		return Spec{}, err
	}
	return Spec{
		OnCalendar:  fmt.Sprintf("*-*-* %s:00", t),
		Description: "daily at " + t,
	}, nil
}

var weekdays = map[string]string{
	"monday": "Mon", "mon": "Mon",
	"tuesday": "Tue", "tue": "Tue",
	"wednesday": "Wed", "wed": "Wed",
	"thursday": "Thu", "thu": "Thu",
	"friday": "Fri", "fri": "Fri",
	"saturday": "Sat", "sat": "Sat",
	"sunday": "Sun", "sun": "Sun",
}

var weekdayNames = map[string]string{
	"Mon": "Monday", "Tue": "Tuesday", "Wed": "Wednesday", "Thu": "Thursday",
	"Fri": "Friday", "Sat": "Saturday", "Sun": "Sunday",
}

// Weekly builds a schedule for one or more days of the week (comma-separated
// names like "saturday" or "sat,sun") at the given time.
func Weekly(days, at string) (Spec, error) {
	if at == "" {
		return Spec{}, fmt.Errorf("--weekly needs a time — add --at HH:MM, e.g. --weekly %s --at 03:00", days)
	}
	t, err := ParseTime(at)
	if err != nil {
		return Spec{}, err
	}

	var abbrs, names []string
	seen := map[string]bool{}
	for _, raw := range strings.Split(days, ",") {
		day := strings.ToLower(strings.TrimSpace(raw))
		if day == "" {
			continue
		}
		abbr, ok := weekdays[day]
		if !ok {
			return Spec{}, fmt.Errorf("unknown day %q — use monday..sunday or mon..sun", raw)
		}
		if !seen[abbr] {
			seen[abbr] = true
			abbrs = append(abbrs, abbr)
			names = append(names, weekdayNames[abbr])
		}
	}
	if len(abbrs) == 0 {
		return Spec{}, fmt.Errorf("no day given — e.g. --weekly saturday or --weekly sat,sun")
	}

	return Spec{
		OnCalendar:  fmt.Sprintf("%s *-*-* %s:00", strings.Join(abbrs, ","), t),
		Description: fmt.Sprintf("weekly on %s at %s", strings.Join(names, ", "), t),
	}, nil
}

// Monthly builds a schedule for a day of month (1-28) or "last" (the final
// day of every month, however long the month is) at the given time.
// Days 29-31 are rejected: systemd would silently skip months that don't
// have them, and a backup that skips February is a trap.
func Monthly(day, at string) (Spec, error) {
	if at == "" {
		return Spec{}, fmt.Errorf("--monthly needs a time — add --at HH:MM, e.g. --monthly %s --at 03:00", day)
	}
	t, err := ParseTime(at)
	if err != nil {
		return Spec{}, err
	}

	if strings.EqualFold(day, "last") {
		return Spec{
			OnCalendar:  fmt.Sprintf("*-*-~1 %s:00", t),
			Description: "monthly on the last day of the month at " + t,
		}, nil
	}
	n, err := strconv.Atoi(day)
	if err != nil {
		return Spec{}, fmt.Errorf("invalid day of month %q — use 1-28 or \"last\"", day)
	}
	if n >= 29 && n <= 31 {
		return Spec{}, fmt.Errorf("day %d doesn't exist in every month, so some months would be skipped — use --monthly last for the last day of the month", n)
	}
	if n < 1 || n > 28 {
		return Spec{}, fmt.Errorf("day of month must be 1-28 or \"last\", got %d", n)
	}
	return Spec{
		OnCalendar:  fmt.Sprintf("*-*-%02d %s:00", n, t),
		Description: fmt.Sprintf("monthly on day %d at %s", n, t),
	}, nil
}

func serviceUnit(s Spec) string {
	return fmt.Sprintf(`[Unit]
Description=boxdb upload to S3
Wants=network-online.target
After=network-online.target

[Service]
Type=oneshot
User=%s
ExecStart=%s upload
`, s.User, s.Exec)
}

func timerUnit(s Spec) string {
	return fmt.Sprintf(`[Unit]
Description=%s%s

[Timer]
OnCalendar=%s
Persistent=true

[Install]
WantedBy=timers.target
`, descPrefix, s.Description, s.OnCalendar)
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
func Install(s Spec) error {
	if err := requireSystemd(); err != nil {
		return err
	}
	if os.Geteuid() != 0 {
		return fmt.Errorf("writing to %s needs root — rerun with sudo", unitDir)
	}
	// Let systemd double-check the calendar expression before we commit it.
	if _, err := exec.LookPath("systemd-analyze"); err == nil {
		if out, err := exec.Command("systemd-analyze", "calendar", s.OnCalendar).CombinedOutput(); err != nil {
			return fmt.Errorf("systemd rejected calendar expression %q: %s", s.OnCalendar, strings.TrimSpace(string(out)))
		}
	}
	if err := os.WriteFile(filepath.Join(unitDir, serviceFile), []byte(serviceUnit(s)), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(unitDir, timerFile), []byte(timerUnit(s)), 0o644); err != nil {
		return err
	}
	if err := systemctl("daemon-reload"); err != nil {
		return err
	}
	return systemctl("enable", "--now", timerFile)
}

// NextElapse asks systemd when the expression fires next. Best effort:
// returns "" when it can't tell.
func NextElapse(expr string) string {
	out, err := exec.Command("systemd-analyze", "calendar", expr).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(line), "Next elapse:"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
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
	Schedule   string // human description, e.g. "weekly on Saturday at 03:00"
	OnCalendar string
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
			if v, ok := strings.CutPrefix(line, "Description="+descPrefix); ok {
				info.Schedule = v
			}
			if v, ok := strings.CutPrefix(line, "OnCalendar="); ok {
				info.OnCalendar = v
			}
			// Timers installed before v0.8.0 carry the time in OnCalendar only.
			if info.Schedule == "" {
				if v, ok := strings.CutPrefix(line, "OnCalendar=*-*-* "); ok {
					info.Schedule = "daily at " + strings.TrimSuffix(v, ":00")
				}
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
