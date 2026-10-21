package logger

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ============================================================
// 统一运行日志：
//   - 级别控制（debug / info / warn / error），对应 config.log_level
//   - 同时输出到控制台(stderr)与文件 <data_dir>/logs/moyue.log
//   - 文件按大小自动轮转（默认 20MB，保留最近 3 份），防止无限膨胀
//   - 使用标准库 log/slog 结构化输出；旧代码的 log.* 调用经 SetOutput 一并落盘
// ============================================================

const (
	DefaultFileName = "moyue.log"
	defaultMaxMB    = 20
	defaultBackups  = 3
)

var (
	mu          sync.Mutex
	initialized bool
	fileW       *rotateFileWriter
	multiW      io.Writer
)

// preWriter 程序启动早期（包加载 ~ Init）的日志缓冲：
// 立即回显到 stderr（与控制台行为一致），同时缓存内容，待 Init 确定日志目录后一并写入文件，
// 避免 config.Load 等早期诊断日志丢失。
type preWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

var pre = &preWriter{}

func init() {
	log.SetOutput(pre)
}

func (w *preWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	_, _ = os.Stderr.Write(p)
	_, _ = w.buf.Write(p)
	return len(p), nil
}

func (w *preWriter) flushTo(out io.Writer) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.buf.Len() > 0 {
		_, _ = out.Write(w.buf.Bytes())
		w.buf.Reset()
	}
}

// Init 初始化运行日志。level 取值：debug/info/warn/error（非法值回退 info）；
// dir 为日志目录（不存在时自动创建），日志文件为 <dir>/moyue.log。
func Init(level, dir string) error {
	mu.Lock()
	defer mu.Unlock()

	lv := parseLevel(level)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fallbackConsole()
		return fmt.Errorf("logger: create log dir %s: %w", dir, err)
	}
	fw, err := newRotateFileWriter(filepath.Join(dir, DefaultFileName), defaultMaxMB, defaultBackups)
	if err != nil {
		fallbackConsole()
		return fmt.Errorf("logger: open log file: %w", err)
	}
	fileW = fw
	multiW = io.MultiWriter(os.Stderr, fw)

	// 冲刷启动早期缓存的日志并接管标准库 log 输出
	pre.flushTo(multiW)
	log.SetOutput(multiW)

	// slog：结构化输出，时间/级别/消息 + 附加字段
	h := slog.NewTextHandler(multiW, &slog.HandlerOptions{Level: lv})
	slog.SetDefault(slog.New(h))
	initialized = true
	slog.Info("运行日志已初始化", "level", lv.String(), "file", fw.path)
	return nil
}

// fallbackConsole 初始化失败时：早期缓冲与标准 log 回退到控制台输出
func fallbackConsole() {
	pre.flushTo(os.Stderr)
	log.SetOutput(os.Stderr)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Writer 返回同时写入控制台与日志文件的 writer，供 gin 及第三方日志接入。
// Init 前调用返回 stderr。
func Writer() io.Writer {
	mu.Lock()
	defer mu.Unlock()
	if multiW == nil {
		return os.Stderr
	}
	return multiW
}

// ===== slog 便捷函数 =====

func Debug(msg string, args ...any) { slog.Debug(msg, args...) }
func Info(msg string, args ...any)  { slog.Info(msg, args...) }
func Warn(msg string, args ...any)  { slog.Warn(msg, args...) }
func Error(msg string, args ...any) { slog.Error(msg, args...) }

func Debugf(format string, a ...any) { slog.Debug(fmt.Sprintf(format, a...)) }
func Infof(format string, a ...any)  { slog.Info(fmt.Sprintf(format, a...)) }
func Warnf(format string, a ...any)  { slog.Warn(fmt.Sprintf(format, a...)) }
func Errorf(format string, a ...any) { slog.Error(fmt.Sprintf(format, a...)) }

// ===== 轮转文件 writer =====

type rotateFileWriter struct {
	mu      sync.Mutex
	path    string
	f       *os.File
	maxSize int64
	backups int
}

func newRotateFileWriter(path string, maxMB, backups int) (*rotateFileWriter, error) {
	w := &rotateFileWriter{
		path:    path,
		maxSize: int64(maxMB) << 20,
		backups: backups,
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	w.f = f
	return w, nil
}

func (w *rotateFileWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if st, err := w.f.Stat(); err == nil && st.Size()+int64(len(p)) > w.maxSize {
		if err := w.rotateLocked(); err != nil {
			return 0, err
		}
	}
	return w.f.Write(p)
}

// rotateLocked 轮转：moyue.log -> moyue.log.1 -> .2 -> .3，删除最旧备份后重开新文件
func (w *rotateFileWriter) rotateLocked() error {
	if err := w.f.Close(); err != nil {
		return err
	}
	if w.backups > 0 {
		_ = os.Remove(fmt.Sprintf("%s.%d", w.path, w.backups))
		for i := w.backups - 1; i >= 1; i-- {
			_ = os.Rename(fmt.Sprintf("%s.%d", w.path, i), fmt.Sprintf("%s.%d", w.path, i+1))
		}
		_ = os.Rename(w.path, w.path+".1")
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	w.f = f
	return nil
}
