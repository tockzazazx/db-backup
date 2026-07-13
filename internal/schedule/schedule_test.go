package schedule

import (
	"strings"
	"testing"
)

func TestParseTime(t *testing.T) {
	for in, want := range map[string]string{
		"03:00": "03:00",
		"3:05":  "03:05",
		"23:59": "23:59",
	} {
		got, err := ParseTime(in)
		if err != nil || got != want {
			t.Errorf("ParseTime(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "24:00", "9pm", "12:60", "0300"} {
		if _, err := ParseTime(bad); err == nil {
			t.Errorf("ParseTime(%q) should fail", bad)
		}
	}
}

func TestDaily(t *testing.T) {
	s, err := Daily("03:00")
	if err != nil {
		t.Fatal(err)
	}
	if s.OnCalendar != "*-*-* 03:00:00" {
		t.Errorf("OnCalendar = %q", s.OnCalendar)
	}
	if s.Description != "daily at 03:00" {
		t.Errorf("Description = %q", s.Description)
	}
	if _, err := Daily("25:00"); err == nil {
		t.Error("Daily(25:00) should fail")
	}
}

func TestWeekly(t *testing.T) {
	s, err := Weekly("saturday", "03:00")
	if err != nil {
		t.Fatal(err)
	}
	if s.OnCalendar != "Sat *-*-* 03:00:00" {
		t.Errorf("OnCalendar = %q", s.OnCalendar)
	}
	if s.Description != "weekly on Saturday at 03:00" {
		t.Errorf("Description = %q", s.Description)
	}

	s, err = Weekly("SAT, sun, sat", "9:05") // mixed case, spaces, duplicate
	if err != nil {
		t.Fatal(err)
	}
	if s.OnCalendar != "Sat,Sun *-*-* 09:05:00" {
		t.Errorf("OnCalendar = %q", s.OnCalendar)
	}
	if s.Description != "weekly on Saturday, Sunday at 09:05" {
		t.Errorf("Description = %q", s.Description)
	}

	for _, tc := range []struct{ days, at string }{
		{"saturady", "03:00"}, // typo
		{"saturday", ""},      // missing --at
		{"", "03:00"},         // no day
		{"sat", "24:30"},      // bad time
	} {
		if _, err := Weekly(tc.days, tc.at); err == nil {
			t.Errorf("Weekly(%q, %q) should fail", tc.days, tc.at)
		}
	}
}

func TestMonthly(t *testing.T) {
	s, err := Monthly("1", "03:00")
	if err != nil {
		t.Fatal(err)
	}
	if s.OnCalendar != "*-*-01 03:00:00" {
		t.Errorf("OnCalendar = %q", s.OnCalendar)
	}
	if s.Description != "monthly on day 1 at 03:00" {
		t.Errorf("Description = %q", s.Description)
	}

	s, err = Monthly("last", "14:30")
	if err != nil {
		t.Fatal(err)
	}
	if s.OnCalendar != "*-*-~1 14:30:00" {
		t.Errorf("OnCalendar = %q", s.OnCalendar)
	}
	if !strings.Contains(s.Description, "last day of the month") {
		t.Errorf("Description = %q", s.Description)
	}

	for _, tc := range []struct{ day, at string }{
		{"29", "03:00"}, {"30", "03:00"}, {"31", "03:00"}, // not in every month
		{"0", "03:00"}, {"32", "03:00"}, {"abc", "03:00"},
		{"15", ""}, // missing --at
	} {
		if _, err := Monthly(tc.day, tc.at); err == nil {
			t.Errorf("Monthly(%q, %q) should fail", tc.day, tc.at)
		}
	}
}

func TestUnits(t *testing.T) {
	s, _ := Weekly("sat", "03:00")
	s.User = "user01"
	s.Exec = "/usr/bin/boxdb"

	svc := serviceUnit(s)
	for _, want := range []string{"User=user01", "ExecStart=/usr/bin/boxdb upload", "Type=oneshot", "After=network-online.target"} {
		if !strings.Contains(svc, want) {
			t.Errorf("service unit missing %q:\n%s", want, svc)
		}
	}

	tmr := timerUnit(s)
	for _, want := range []string{
		"Description=boxdb upload weekly on Saturday at 03:00",
		"OnCalendar=Sat *-*-* 03:00:00",
		"Persistent=true",
		"WantedBy=timers.target",
	} {
		if !strings.Contains(tmr, want) {
			t.Errorf("timer unit missing %q:\n%s", want, tmr)
		}
	}
}
