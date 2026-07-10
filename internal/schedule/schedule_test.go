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

func TestUnits(t *testing.T) {
	d := Daily{Time: "03:00", User: "user01", Exec: "/usr/bin/boxdb"}

	svc := serviceUnit(d)
	for _, want := range []string{"User=user01", "ExecStart=/usr/bin/boxdb upload", "Type=oneshot", "After=network-online.target"} {
		if !strings.Contains(svc, want) {
			t.Errorf("service unit missing %q:\n%s", want, svc)
		}
	}

	tmr := timerUnit(d)
	for _, want := range []string{"OnCalendar=*-*-* 03:00:00", "Persistent=true", "WantedBy=timers.target"} {
		if !strings.Contains(tmr, want) {
			t.Errorf("timer unit missing %q:\n%s", want, tmr)
		}
	}
}
