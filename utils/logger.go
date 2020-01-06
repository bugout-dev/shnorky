package utils

import (
	"os"

	"github.com/sirupsen/logrus"
)

// LogLevels - mapping between log level specification strings and logrus Level values
var LogLevels = map[string]logrus.Level{
	"TRACE": logrus.TraceLevel,
	"DEBUG": logrus.DebugLevel,
	"INFO":  logrus.InfoLevel,
	"WARN":  logrus.WarnLevel,
	"ERROR": logrus.ErrorLevel,
	"FATAL": logrus.FatalLevel,
	"PANIC": logrus.PanicLevel,
}

// Logger creates a new simplex logger with parameters specified by environment variables.
// Accepts the following environment variables:
// + LOG_LEVEL (value should be one of TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC)
func Logger() *logrus.Logger {
	log := logrus.New()
	rawLevel := os.Getenv("LOG_LEVEL")
	if rawLevel == "" {
		rawLevel = "ERROR"
	}
	level, ok := LogLevels[rawLevel]
	if !ok {
		log.Fatalf("Invalid value for LOG_LEVEL environment variable: %s. Choose one of TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC", rawLevel)
	}
	log.SetLevel(level)

	return log
}
