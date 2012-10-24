package skynet

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogPayload stores detailed logging information, including all
// fields used by Clarity Service's semantic_logger (see
// https://github.com/ClarityServices/semantic_logger). See the
// LogPayload struct's inline comments for instructions as to which
// fields should be populated by whom or what (e.g., the user
// (manually), helper functions, or methods on the various loggers in
// this package.)
// Valid log levels -- a list of which is stored in the `LogLevels`
// slice -- include TRACE, DEBUG, INFO, WARN, ERROR, and FATAL.
type LogPayload struct {
	// Set by user by passing values to NewLogPayload()
	Level   LogLevel `json:"level"`
	Message string   `json:"message"`
	// Set automatically within NewLogPayload()
	Action string `json:"action"`
	// Set by .setKnownFields()
	Application string    `json:"application"`
	PID         int       `json:"pid"`
	Time        time.Time `json:"time"`
	HostName    string    `json:"host_name"`
	// Set by .SetTags() convenience method
	Tags []string `json:"tags"`
	// Should be set by .Log()
	Name  string `json:"name"`  // Store "class name" (type)
	UUID  string `json:"uuid"`  // Logger's UUID
	Table string `json:"table"` // Mongo collection name
	// Set by Fatal() method if need be
	Backtrace []string `json:"backtrace"`
	// Should be set by BenchmarkInfo() if called
	Duration time.Duration `json:"duration"`
	// Optionally set by user manually
	ThreadName string `json:"thread_name"`
}

// Exception formats the payload just as
// github.com/ClarityServices/semantic_logger formats Exceptions for
// logging. This package has no Exception data type; all relevant data
// should be stored in a *LogPayload. The payload's "exception" data is
// generated from a panic's stacktrace using the `genStacktrace`
// helper function.
func (payload *LogPayload) Exception() string {
	// message << " -- " << "#{exception.class}: #{exception.message}\n
	// #{(exception.backtrace || []).join("\n")}"
	formatStr := "%s -- %s: %s\n%s"
	backtrace := strings.Join(payload.Backtrace, "\n")
	return fmt.Sprintf(formatStr, payload.Message, "panic",
		payload.Message, backtrace)
}

// setKnownFields sets the `Application`, `PID`, `Time`, and
// `HostName` fields of the given payload. See the documentation on
// the LogPayload type for which fields should be set where, and by
// whom (the user) or what (a function or method).
func (payload *LogPayload) setKnownFields() {
	// Set Application to os.Args[0] if it wasn't set by the user
	if payload.Application == "" {
		payload.Application = os.Args[0]
	}
	payload.PID = os.Getpid()
	payload.Time = time.Now()
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("Error getting hostname: %v\n", err)
	}
	payload.HostName = hostname
}

// SetTags is a convenience method for adding tags to *LogPayload's,
// since `payload.SetTags("tag1", "tag2")` is cleaner than
// `payload.Tags = []string{"tag1", "tag2"}`
func (payload *LogPayload) SetTags(tags ...string) {
	payload.Tags = tags
}

// NewLogPayload is a convenience function for creating *LogPayload's
func NewLogPayload(level LogLevel, formatStr string,
	vars ...interface{}) *LogPayload {

	payload := &LogPayload{
		Level:   level,
		Message: fmt.Sprintf(formatStr, vars...),
		// TODO: Make sure that `2` is the number that should be
		// passed in here
		Action: getCallerName(2),
	}
	// payload.setKnownFields() called in .Log() method; not calling here

	// TODO: Come up with a way to intelligently auto-fill ThreadName,
	// if possible

	return payload
}

func getCallerName(skip int) string {
	pc, _, _, _ := runtime.Caller(skip)
	f := runtime.FuncForPC(pc)
	return f.Name()
}

// LogLevels are ints for the sake of having a well-defined
// ordering. This is useful for viewing logs more or less severe than
// a given log level. See the LogLevel.LessSevereThan method.
type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

// LogLevel stores the valid log levels as specified by
// github.com/ClarityServices/semantic_logger.
var LogLevels = []LogLevel{
	TRACE, DEBUG, INFO, WARN, ERROR, FATAL,
}

// LessSevereThan tells you whether or not `level` is a less severe
// LogLevel than `level2`. This is useful for determining which logs
// to view.
func (level LogLevel) LessSevereThan(level2 LogLevel) bool {
	return int(level) < int(level2)
}

// String helps make LogLevel's more readable by representing them as
// strings instead of ints 0 (TRACE) through 5 (FATAL).
func (level LogLevel) String() string {
	switch level {
	case TRACE:
		return "TRACE"
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	}
	return "CUSTOM"
}

// NOTE: The data type names are what they are (and rather verbose) in
// part so that "SemanticLogger" can be replaced with "Logger" in this
// file once the contents of logger.go is no longer needed.

// SemanticLogger is meant to match the format and functionality of
// github.com/ClarityServices/semantic_logger
type SemanticLogger interface {
	Log(payload *LogPayload)
	Trace(msg string)
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	Fatal(msg string)
	BenchmarkInfo(level LogLevel, msg string, f func(logger SemanticLogger))
}

// genStacktrace is a helper function for generating stacktrace
// data. Used to populate (*LogPayload).Backtrace
func genStacktrace() (stacktrace []string) {
	// TODO: Make sure that `skip` should begin at 1, not 2
	for skip := 1; ; skip++ {
		pc, file, line, ok := runtime.Caller(skip)
		if !ok {
			break
		}
		f := runtime.FuncForPC(pc)
		traceLine := fmt.Sprintf("%s:%d %s()\n", file, line, f.Name())
		stacktrace = append(stacktrace, traceLine)
	}
	return
}
