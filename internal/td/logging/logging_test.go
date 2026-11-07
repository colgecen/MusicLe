package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetFileAndPrintf(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "log")
	if err := SetFile(path); err != nil {
		t.Fatalf("SetFile: %v", err)
	}
	Printf("hello %s", "world")
	Close()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), "hello world") {
		t.Errorf("log = %q", data)
	}
}

func TestOffAndStderrModes(t *testing.T) {
	if err := SetFile("off"); err != nil {
		t.Fatalf("off: %v", err)
	}
	Printf("must not appear")
	Close()
	if err := SetFile("-"); err != nil {
		t.Fatalf("stderr: %v", err)
	}
	Close()
}

func TestRotatesLargeLog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log")
	if err := os.WriteFile(path, make([]byte, maxSize+1), 0644); err != nil {
		t.Fatal(err)
	}
	if err := SetFile(path); err != nil {
		t.Fatal(err)
	}
	Close()
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf("rotated log missing: %v", err)
	}
}
