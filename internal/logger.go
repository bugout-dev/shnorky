package internal

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

// Accepts the following environment variables:
// + LOG_LEVEL (value should be one of TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC)
func GenerateLogger() *logrus.Logger {
	log := logrus.New()

	rawLevel := os.Getenv("LOG_LEVEL")
	if rawLevel == "" {
		rawLevel = "WARN"
	}
	level, ok := LogLevels[rawLevel]
	if !ok {
		log.Fatalf("Invalid value for LOG_LEVEL environment variable: %s. Choose one of TRACE, DEBUG, INFO, WARN, ERROR, FATAL, PANIC", rawLevel)
	}
	log.SetLevel(level)

	return log
}
