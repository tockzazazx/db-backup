package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitExt(t *testing.T) {
	cases := map[string][2]string{
		"1.pg":           {"1", ".pg"},
		"db.sql":         {"db", ".sql"},
		"backup.tar.gz":  {"backup", ".tar.gz"},
		"backup.TAR.GZ":  {"backup", ".TAR.GZ"},
		"dump.sql.gz":    {"dump", ".sql.gz"},
		"data.tar.zst":   {"data", ".tar.zst"},
		"noext":          {"noext", ""},
		"weird.name.pg":  {"weird.name", ".pg"},
		"archive.tar.xz": {"archive", ".tar.xz"},
	}
	for in, want := range cases {
		base, ext := splitExt(in)
		if base != want[0] || ext != want[1] {
			t.Errorf("splitExt(%q) = (%q, %q); want (%q, %q)", in, base, ext, want[0], want[1])
		}
	}
}

func TestUniquePath(t *testing.T) {
	dir := t.TempDir()

	p, err := uniquePath(dir, "1.pg")
	if err != nil || p != filepath.Join(dir, "1.pg") {
		t.Fatalf("first uniquePath = %q, %v", p, err)
	}
	touch(t, p)

	p, _ = uniquePath(dir, "1.pg")
	if p != filepath.Join(dir, "1 (1).pg") {
		t.Errorf("second uniquePath = %q; want 1 (1).pg", p)
	}
	touch(t, p)

	p, _ = uniquePath(dir, "1.pg")
	if p != filepath.Join(dir, "1 (2).pg") {
		t.Errorf("third uniquePath = %q; want 1 (2).pg", p)
	}

	touch(t, filepath.Join(dir, "backup.tar.gz"))
	p, _ = uniquePath(dir, "backup.tar.gz")
	if p != filepath.Join(dir, "backup (1).tar.gz") {
		t.Errorf("tar.gz uniquePath = %q; want backup (1).tar.gz", p)
	}
}

func touch(t *testing.T, p string) {
	t.Helper()
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}
