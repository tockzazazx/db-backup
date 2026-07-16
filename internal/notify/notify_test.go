package notify

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestReceiverList(t *testing.T) {
	cases := map[string][]string{
		"{'a@x.com'}":            {"a@x.com"},
		"{'a@x.com','b@y.com'}":  {"a@x.com", "b@y.com"},
		"{'a@x.com','b@y.com'} ": nil, // placeholder, replaced below
	}
	delete(cases, "{'a@x.com','b@y.com'} ")

	for want, in := range cases {
		if got := receiverList(in); got != want {
			t.Errorf("receiverList(%v) = %q; want %q", in, got, want)
		}
	}

	// Spaces and empties are cleaned up.
	if got := receiverList([]string{" a@x.com ", "", "b@y.com"}); got != "{'a@x.com','b@y.com'}" {
		t.Errorf("receiverList with spaces = %q", got)
	}
}

func TestReportContent(t *testing.T) {
	rep := Report{
		Host:     "PRD-test",
		Dest:     "ubuntu-tock/2026-07-16",
		Started:  time.Date(2026, 7, 16, 3, 0, 0, 0, time.UTC),
		Duration: 12 * time.Second,
		Uploaded: []File{{Name: "db1.pg", Size: 300 * 1024}},
		Skipped:  2,
	}

	if s := rep.Subject(); !strings.Contains(s, "สำเร็จ") || !strings.Contains(s, "PRD-test") {
		t.Errorf("success subject = %q", s)
	}
	c := rep.Content()
	for _, want := range []string{"PRD-test", "ubuntu-tock/2026-07-16", "db1.pg", "300.0 KB", "2 ไฟล์"} {
		if !strings.Contains(c, want) {
			t.Errorf("content missing %q:\n%s", want, c)
		}
	}

	rep.Err = errors.New("connection refused")
	if s := rep.Subject(); !strings.Contains(s, "ล้มเหลว") {
		t.Errorf("failure subject = %q", s)
	}
	if c := rep.Content(); !strings.Contains(c, "connection refused") {
		t.Errorf("failure content missing error:\n%s", c)
	}
}
