// Package logging provides structured logging for dtiam using logrus.
// All log output goes to stderr so it doesn't interfere with stdout data.
package logging

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// Log is the package-level logger used throughout dtiam.
var Log = logrus.New()

func init() {
	Log.SetOutput(os.Stderr)
	Log.SetLevel(logrus.WarnLevel)
	Log.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
}

// Init configures the logger based on verbosity level.
// 0 = warn (default), 1 = info, 2+ = debug.
func Init(verbosity int) {
	switch {
	case verbosity >= 2:
		Log.SetLevel(logrus.DebugLevel)
		Log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	case verbosity == 1:
		Log.SetLevel(logrus.InfoLevel)
	default:
		Log.SetLevel(logrus.WarnLevel)
	}
}

// SetOutput changes the logger's output writer (useful for testing).
func SetOutput(w io.Writer) {
	Log.SetOutput(w)
}

// WithField returns a log entry with a single field.
func WithField(key string, value any) *logrus.Entry {
	return Log.WithField(key, value)
}

// WithFields returns a log entry with multiple fields.
func WithFields(fields map[string]any) *logrus.Entry {
	return Log.WithFields(logrus.Fields(fields))
}

// Debug logs a debug message.
func Debug(args ...any) {
	Log.Debug(args...)
}

// Debugf logs a formatted debug message.
func Debugf(format string, args ...any) {
	Log.Debugf(format, args...)
}

// Info logs an info message.
func Info(args ...any) {
	Log.Info(args...)
}

// Infof logs a formatted info message.
func Infof(format string, args ...any) {
	Log.Infof(format, args...)
}

// Warn logs a warning message.
func Warn(args ...any) {
	Log.Warn(args...)
}

// Warnf logs a formatted warning message.
func Warnf(format string, args ...any) {
	Log.Warnf(format, args...)
}

// Error logs an error message.
func Error(args ...any) {
	Log.Error(args...)
}

// Errorf logs a formatted error message.
func Errorf(format string, args ...any) {
	Log.Errorf(format, args...)
}

// HTTPRequest logs an HTTP request at debug level.
func HTTPRequest(method, url string, statusCode int) {
	Log.WithFields(logrus.Fields{
		"method": method,
		"url":    url,
		"status": statusCode,
	}).Debug("HTTP request")
}

// HTTPRequestStart logs the start of an HTTP request at debug level.
func HTTPRequestStart(method, url string) {
	Log.WithFields(logrus.Fields{
		"method": method,
		"url":    url,
	}).Debug("HTTP request start")
}
