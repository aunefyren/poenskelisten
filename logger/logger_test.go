package logger

import (
	"aunefyren/poenskelisten/models"
	"os"
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
