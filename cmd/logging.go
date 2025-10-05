package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

const formatText = "text"
const formatJSON = "json"

// initLogging initializes an embedded slog Logger.
func initLogging(logLevelStr string, format string) error {
	level := slog.LevelInfo

	if err := level.UnmarshalText([]byte(logLevelStr)); err != nil {
		return err
	}
	logFile := os.Stderr
	// TODO - add support for file logging

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
		ReplaceAttr: func(groups []string, attr slog.Attr) slog.Attr {
			// Shorten file path for source logging
			if attr.Key == slog.SourceKey {
				if source, ok := attr.Value.Any().(*slog.Source); ok {
					source.File = filepath.Base(source.File)
				}
			}
			return attr
		},
	}

	switch format {
	case formatJSON:
		slog.SetDefault(slog.New(slog.NewJSONHandler(logFile, opts)))
	case formatText:
		cw := newLevelColorWriter(logFile, defaultColorEnabled())
		slog.SetDefault(slog.New(slog.NewTextHandler(cw, opts)))
	default:
		return fmt.Errorf("invalid log format: %q", format)
	}
	return nil
}

// levelColorWriter wraps an io.Writer and injects ANSI color codes
// around the value of the `level=...` token in each log line.
// It buffers until '\n' so it always rewrites whole records.
type levelColorWriter struct {
	mu  sync.Mutex
	dst io.Writer
	buf bytes.Buffer
	// enable allows you to disable color when not a TTY.
	enable bool
}

func newLevelColorWriter(dst io.Writer, enable bool) *levelColorWriter {
	return &levelColorWriter{dst: dst, enable: enable}
}

func (w *levelColorWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Accumulate and flush per line.
	total := len(p)
	for len(p) > 0 {
		i := bytes.IndexByte(p, '\n')
		if i == -1 {
			_, _ = w.buf.Write(p)
			break
		}
		// Write up to including newline
		w.buf.Write(p[:i+1])
		line := w.buf.Bytes()
		if w.enable {
			line = colorizeLevel(line)
		}
		if _, err := w.dst.Write(line); err != nil {
			// drop buffered data on error
			w.buf.Reset()
			return 0, err
		}
		w.buf.Reset()
		p = p[i+1:]
	}
	return total, nil
}

const (
	cReset = "\x1b[0m"
	cRed   = "\x1b[31m"
	cYel   = "\x1b[33m"
	cGrn   = "\x1b[32m"
	cCyn   = "\x1b[36m"
	cBlu   = "\x1b[34m"
	cMag   = "\x1b[35m"
)

// colorizeLevel finds `level=XYZ` and wraps XYZ in ANSI color.
// It keeps the rest of the line intact (time, msg, attrs).
func colorizeLevel(line []byte) []byte {
	const key = "level="
	pos := bytes.Index(line, []byte(key))
	if pos < 0 {
		return line
	}
	start := pos + len(key)
	// value is contiguous non-space bytes after "level="
	end := start
	for end < len(line) && line[end] != ' ' && line[end] != '\n' && line[end] != '\r' {
		end++
	}
	lv := string(line[start:end])

	color := ""
	switch lv {
	case "DEBUG":
		color = cCyn
	case "INFO":
		color = cGrn
	case "WARN", "WARNING":
		color = cYel
	case "ERROR":
		color = cRed
	case "TRACE":
		color = cBlu
	case "CRIT", "FATAL":
		color = cMag
	default:
		// unknown or custom levels; leave uncolored
		return line
	}

	var out bytes.Buffer
	out.Grow(len(line) + 10)
	out.Write(line[:start])
	out.WriteString(color)
	out.Write(line[start:end])
	out.WriteString(cReset)
	out.Write(line[end:])
	return out.Bytes()
}

// defaultColorEnabled implements a simple TTY gate. You can swap in mattn/go-isatty if you like.
func defaultColorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	// crude: assume stderr is a terminal if TERM looks sane
	term := os.Getenv("TERM")
	return term != "" && term != "dumb"
}
