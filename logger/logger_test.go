package logger

import (
	"aunefyren/poenskelisten/models"
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

// InitLogger writes to "files/poenskelisten.log" relative to the working
// directory (logger/ under `go test`), and calls logrus.Fatalf (os.Exit) if it
// can't open that file — so every test here must ensure "files/" exists first,
// and none can exercise that Fatalf path without killing the test binary.
func ensureLogFilesDir(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll("files", 0755); err != nil {
		t.Fatalf("failed to create files dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll("files") })
}

func TestInitLoggerValidLevel(t *testing.T) {
	ensureLogFilesDir(t)

	InitLogger(models.ConfigStruct{PoenskelistenLogLevel: "debug"})

	if Log == nil {
		t.Fatal("expected Log to be initialized")
	}
	if Log.GetLevel() != logrus.DebugLevel {
		t.Errorf("level = %v, want debug", Log.GetLevel())
	}
	if Log.Formatter == nil {
		t.Error("expected a formatter to be set")
	}
}

func TestInitLoggerInvalidLevelFallsBackToInfo(t *testing.T) {
	ensureLogFilesDir(t)

	InitLogger(models.ConfigStruct{PoenskelistenLogLevel: "not-a-real-level"})

	if Log.GetLevel() != logrus.InfoLevel {
		t.Errorf("level = %v, want the info fallback", Log.GetLevel())
	}
}

func TestInitLoggerEmptyLevelFallsBackToInfo(t *testing.T) {
	ensureLogFilesDir(t)

	InitLogger(models.ConfigStruct{})

	if Log.GetLevel() != logrus.InfoLevel {
		t.Errorf("level = %v, want the info fallback for an empty level string", Log.GetLevel())
	}
}

func TestEscapeControlCharacters(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "user logged in", "user logged in"},
		{"tab kept", "a\tb", "a\tb"},
		{"lf", "a\nb", `a\nb`},
		{"crlf", "a\r\nb", `a\r\nb`},
		{"other c0", "a\x1bb\x00", `a\u001bb\u0000`},
		{"c1 nel", "a\u0085b", `a\u0085b`},
		{"line separator", "a\u2028b\u2029", `a\u2028b\u2029`},
		{"non-ascii kept", "Pønskelisten", "Pønskelisten"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := escapeControlCharacters(c.in); got != c.want {
				t.Errorf("escapeControlCharacters(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// A caller-controlled value containing CRLF must not be able to start a new,
// forged log entry (GHSA-pmhf-h957-p2v3).
func TestFormatterPreventsLogInjection(t *testing.T) {
	var out bytes.Buffer
	log := logrus.New()
	log.SetOutput(&out)
	log.Formatter = newFormatter()

	log.Error("Invalid mime type for image. Type: image/png\r\n[INFO]: 2026-11-03 14:03:16 - User admin successfully authenticated from 127.0.0.1")

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("got %d log lines, want 1: %q", len(lines), out.String())
	}
	if !strings.HasPrefix(lines[0], "[ERROR]: ") || !strings.Contains(lines[0], `image/png\r\n[INFO]`) {
		t.Errorf("log line = %q, want one error entry with the CRLF escaped", lines[0])
	}
	if strings.Contains(out.String(), "\r") {
		t.Errorf("log output contains a raw carriage return: %q", out.String())
	}
}

// The formatter must not mutate the entry other hooks/formatters may still see.
func TestFormatterLeavesEntryUntouched(t *testing.T) {
	entry := logrus.NewEntry(logrus.New())
	entry.Message = "a\nb"
	if _, err := newFormatter().Format(entry); err != nil {
		t.Fatalf("Format: %v", err)
	}
	if entry.Message != "a\nb" {
		t.Errorf("entry.Message = %q, want it unchanged", entry.Message)
	}
}
