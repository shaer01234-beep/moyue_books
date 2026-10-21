package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitWritesLogFile(t *testing.T) {
	dir := t.TempDir()
	if err := Init("info", dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	Infof("hello %s", "world")
	Warn("something warn")

	data, err := os.ReadFile(filepath.Join(dir, DefaultFileName))
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "hello world") {
		t.Errorf("log file missing info line: %q", s)
	}
	if !strings.Contains(s, "something warn") {
		t.Errorf("log file missing warn line: %q", s)
	}
}

func TestLevelFilter(t *testing.T) {
	dir := t.TempDir()
	if err := Init("warn", dir); err != nil {
		t.Fatalf("Init: %v", err)
	}
	Debug("should be filtered")
	Error("error visible")

	data, err := os.ReadFile(filepath.Join(dir, DefaultFileName))
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	s := string(data)
	if strings.Contains(s, "should be filtered") {
		t.Errorf("debug line leaked at warn level: %q", s)
	}
	if !strings.Contains(s, "error visible") {
		t.Errorf("error line missing: %q", s)
	}
}

func TestRotate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	w, err := newRotateFileWriter(path, 1, 3) // 1MB
	if err != nil {
		t.Fatalf("newRotateFileWriter: %v", err)
	}
	// 写入 3.5MB 数据，触发至少 3 次轮转
	chunk := strings.Repeat("a", 256*1024)
	for i := 0; i < 14; i++ {
		if _, err := w.Write([]byte(chunk)); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	// 当前文件存在
	if _, err := os.Stat(path); err != nil {
		t.Errorf("current log missing: %v", err)
	}
	// 至少存在 .1 备份
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf("backup .1 missing: %v", err)
	}
	// 最旧备份不应超过 3
	if _, err := os.Stat(path + ".4"); err == nil {
		t.Errorf("backup .4 should have been removed")
	}
}
