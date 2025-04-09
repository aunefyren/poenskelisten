package logger

import (
	"aunefyren/poenskelisten/models"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func InitLogger(configFile models.ConfigStruct) {
	Log = logrus.New()

	// Define log file
	logFile, err := os.OpenFile("files/poenskelisten.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logrus.Fatalf("Failed to load log file: %v", err)
	}

	// Output to both stdout and log file
	mw := io.MultiWriter(os.Stdout, logFile)
	Log.SetOutput(mw)

	// Set log level
	level, err := logrus.ParseLevel(configFile.PoenskelistenLogLevel)
	if err != nil {
		logrus.Error("Failed to load log file: %v", err)
		level = logrus.InfoLevel
	}

	Log.SetLevel(level)

	Log.Info("Log level set to: " + level.String())
}
