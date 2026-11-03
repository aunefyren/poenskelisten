package logger

import (
	"aunefyren/poenskelisten/models"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var Log *logrus.Logger

func InitLogger(configFile models.ConfigStruct) {
	Log = logrus.New()

	// Define log file
	logFile, err := os.OpenFile("files/poenskelisten.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logrus.Fatalf("failed to load log file: %v", err)
	}

	Log.Formatter = newFormatter()

	// Output to both stdout and log file
	mw := io.MultiWriter(os.Stdout, logFile)
	Log.SetOutput(mw)

	// Set log level
	level, err := logrus.ParseLevel(configFile.PoenskelistenLogLevel)
	if err != nil {
		logrus.Errorf("failed to load log file: %v", err)
		level = logrus.InfoLevel
	}

	Log.SetLevel(level)

	Log.Info("log level set to: " + level.String())
}

func newFormatter() logrus.Formatter {
	return &escapingFormatter{inner: &easy.Formatter{
		TimestampFormat: "2006-01-02 15:04:05",
		LogFormat:       "[%lvl%]: %time% - %msg%\n",
	}}
}

// escapingFormatter escapes control characters in every message before it is
// formatted. Log lines are one entry per line and routinely carry
// caller-supplied values (e-mail addresses, upload MIME types, error text
// echoing input), so an unescaped CR/LF would let a caller forge entries.
type escapingFormatter struct {
	inner logrus.Formatter
}

func (f *escapingFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	escaped := *entry
	escaped.Message = escapeControlCharacters(entry.Message)
	return f.inner.Format(&escaped)
}

func escapeControlCharacters(message string) string {
	if !strings.ContainsFunc(message, isEscapedRune) {
		return message
	}

	var builder strings.Builder
	for _, r := range message {
		switch {
		case r == '\n':
			builder.WriteString(`\n`)
		case r == '\r':
			builder.WriteString(`\r`)
		case isEscapedRune(r):
			builder.WriteString(fmt.Sprintf(`\u%04x`, r))
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// isEscapedRune covers C0/C1 controls (tab excepted) plus the Unicode line and
// paragraph separators, which some log viewers also render as line breaks.
func isEscapedRune(r rune) bool {
	if r == '\t' {
		return false
	}
	return r < 0x20 || (r >= 0x7f && r <= 0x9f) || r == '\u2028' || r == '\u2029'
}
