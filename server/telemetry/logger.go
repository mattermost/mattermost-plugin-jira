// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

// Copied from pluginapi/experimental/bot/logger to avoid upgrading min_server_version
// remove this file once pluginapi can be updated to 0.1.3+ ( min_server_version is
//
//	safe to point at 7.x)
package telemetry

import (
	"fmt"
	"time"
)

const (
	timed   = "__since"
	elapsed = "Elapsed"

	ErrorKey = "error"
)

// LogLevel defines the level of log messages
type LogLevel string

const (
	// LogLevelDebug denotes debug messages
	LogLevelDebug = "debug"
	// LogLevelInfo denotes info messages
	LogLevelInfo = "info"
	// LogLevelWarn denotes warn messages
	LogLevelWarn = "warn"
	// LogLevelError denotes error messages
	LogLevelError = "error"
)

// LogContext defines the context for the logs.
type LogContext map[string]interface{}

// Logger defines an object able to log messages.
type Logger interface {
	// With adds a logContext to the logger.
	With(LogContext) Logger
	// WithError adds an Error to the logger.
	WithError(error) Logger
	// Context returns the current context
	Context() LogContext
	// Timed add a timed log context.
	Timed() Logger
	// Debugf logs a formatted string as a debug message.
	Debugf(format string, args ...interface{})
	// Errorf logs a formatted string as an error message.
	Errorf(format string, args ...interface{})
	// Infof logs a formatted string as an info message.
	Infof(format string, args ...interface{})
	// Warnf logs a formatted string as an warning message.
	Warnf(format string, args ...interface{})
}

type LogAPI interface {
	LogError(message string, keyValuePairs ...interface{})
	LogWarn(message string, keyValuePairs ...interface{})
	LogInfo(message string, keyValuePairs ...interface{})
	LogDebug(message string, keyValuePairs ...interface{})
}

type defaultLogger struct {
	logContext LogContext
	logAPI     LogAPI
}

func measure(lc LogContext) {
	if lc[timed] == nil {
		return
	}
	started := lc[timed].(time.Time)
	lc[elapsed] = time.Since(started).String()
	delete(lc, timed)
}

func toKeyValuePairs(in map[string]interface{}) (out []interface{}) {
	for k, v := range in {
		out = append(out, k, v)
	}
	return out
}

/*
New creates a new logger.

- api: LogAPI implementation
*/
func NewLogger(api LogAPI) Logger {
	l := &defaultLogger{
		logAPI: api,
	}
	return l
}

func (l *defaultLogger) With(logContext LogContext) Logger {
	newLogger := *l
	if len(newLogger.logContext) == 0 {
		newLogger.logContext = map[string]interface{}{}
	}
	for k, v := range logContext {
		newLogger.logContext[k] = v
	}
	return &newLogger
}

func (l *defaultLogger) WithError(err error) Logger {
	newLogger := *l
	if len(newLogger.logContext) == 0 {
		newLogger.logContext = map[string]interface{}{}
	}
	newLogger.logContext[ErrorKey] = err.Error()
	return &newLogger
}

func (l *defaultLogger) Context() LogContext {
	return l.logContext
}

func (l *defaultLogger) Timed() Logger {
	return l.With(LogContext{
		timed: time.Now(),
	})
}

func (l *defaultLogger) Debugf(format string, args ...interface{}) {
	measure(l.logContext)
	message := fmt.Sprintf(format, args...)
	l.logAPI.LogDebug(message, toKeyValuePairs(l.logContext)...)
}

func (l *defaultLogger) Errorf(format string, args ...interface{}) {
	measure(l.logContext)
	message := fmt.Sprintf(format, args...)
	l.logAPI.LogError(message, toKeyValuePairs(l.logContext)...)
}

func (l *defaultLogger) Infof(format string, args ...interface{}) {
	measure(l.logContext)
	message := fmt.Sprintf(format, args...)
	l.logAPI.LogInfo(message, toKeyValuePairs(l.logContext)...)
}

func (l *defaultLogger) Warnf(format string, args ...interface{}) {
	measure(l.logContext)
	message := fmt.Sprintf(format, args...)
	l.logAPI.LogWarn(message, toKeyValuePairs(l.logContext)...)
}
