// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

// COMPONENT constant for defining the component to identify the log.
const COMPONENT = "component"

// AUDIT constant to identify the audit component.
const AUDIT = "Audit"

// EVENT constant for auditing messages to identify the audit log.
const EVENT = "event"

// AUDITMESSAGE constant for auditing messages to identify the audit log.
const AUDITMESSAGE = "auditmessage"

// USER constant for auditing messages to include the user in the audit log.
const USER = "user"

// EMAIL constant for auditing messages to include the email in the audit log.
const EMAIL = "email"

// OPERATION constant for auditing messages to include the operation in the audit log.
const OPERATION = "operation"

// STATUS constant for auditing messages to include the status in the audit log.
const STATUS = "status"

// ERROR constant for auditing messages to include the error in the audit log.
const ERROR = "error"

// RESPONSE constant for auditing messages to include the response in the audit log.
const RESPONSE = "response"

// PATH constant for auditing messages to include the path in the audit log.
const PATH = "path"

// REQUEST constant for auditing messages to include the request in the audit log.
const REQUEST = "request"

//nolint:gochecknoinits // Using init for defining flags is a valid exception.
func init() {
	flag.Func(
		"globalLogLevel",
		"Sets the application-wide logging level. Must be a valid zerolog.Level. Defaults to 'info'",
		handleLogLevel,
	)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func handleLogLevel(l string) error {
	level, err := zerolog.ParseLevel(l)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)
	return nil
}

type InfraLogger struct {
	zerolog.Logger
}

type InfraCtxLogger struct {
	zerolog.Logger
}

type spanlogHook struct {
	span trace.Span
}

func (h spanlogHook) Run(_ *zerolog.Event, _ zerolog.Level, msg string) {
	if h.span.IsRecording() {
		h.span.AddEvent(msg)
	}
}

func GetLogger(component string) InfraLogger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.TimestampFieldName = "timestamp"
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// use UTC time
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}

	var logger zerolog.Logger
	if _, present := os.LookupEnv("HUMAN"); present {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano})
	} else {
		logger = zerolog.New(os.Stdout)
	}
	if component == AUDIT {
		// Removing internal caller, useless for Audit log.
		logger = logger.With().Timestamp().Str(COMPONENT, component).Logger()
	} else {
		logger = logger.With().Caller().Timestamp().Str(COMPONENT, component).Logger()
	}

	return InfraLogger{logger}
}

func GetLoggerWithCustomWriter(component string, writer *CustomWriter) InfraLogger {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	var loggerCustom zerolog.Logger

	if component == AUDIT {
		// Removing internal caller, useless for Audit log.
		loggerCustom = zerolog.New(writer).With().Timestamp().Str(COMPONENT, component).Logger()
	} else {
		loggerCustom = zerolog.New(writer).With().Caller().Timestamp().Str(COMPONENT, component).Logger()
	}

	return InfraLogger{loggerCustom}
}

func (l InfraLogger) TraceCtx(ctx context.Context) InfraCtxLogger {
	span := trace.SpanFromContext(ctx)
	newlogger := l.With().
		Str("span_id", span.SpanContext().SpanID().String()).
		Str("trace_id", span.SpanContext().TraceID().String()).
		Logger()
	newlogger = newlogger.Hook(spanlogHook{span})
	return InfraCtxLogger{newlogger}
}

// InfraSec is a logging decorator for InfraLogger intended to be used for security related events.
func (l *InfraLogger) InfraSec() *InfraLogger {
	return &InfraLogger{l.With().Str("InfraSec", "true").Logger()}
}

// InfraSec is a logging decorator InfraCtxLogger intended to be used for security related events.
func (l *InfraCtxLogger) InfraSec() *InfraCtxLogger {
	return &InfraCtxLogger{l.With().Str("InfraSec", "true").Logger()}
}

// InfraAuditEvent is a logging decorator for InfraLogger intended to be used for auditing related logs.
func (l *InfraLogger) InfraAuditEvent() *InfraLogger {
	return &InfraLogger{l.With().Str(EVENT, AUDITMESSAGE).Logger()}
}

// InfraAuditEvent is a logging decorator InfraCtxLogger intended to be used for auditing related logs.
func (l *InfraCtxLogger) InfraAuditEvent() *InfraCtxLogger {
	return &InfraCtxLogger{l.With().Str(EVENT, AUDITMESSAGE).Logger()}
}

// InfraAuditUsr is a logging decorator for InfraLogger intended to be used for auditing related logs.
func (l *InfraLogger) InfraAuditUsr(user string) *InfraLogger {
	return &InfraLogger{l.With().Str(USER, user).Logger()}
}

// InfraAuditUsr is a logging decorator InfraCtxLogger intended to be used for auditing related logs.
func (l *InfraCtxLogger) InfraAuditUsr(user string) *InfraCtxLogger {
	return &InfraCtxLogger{l.With().Str(USER, user).Logger()}
}

// InfraAuditEmail is a logging decorator for InfraLogger intended to be used for auditing related logs.
func (l *InfraLogger) InfraAuditEmail(email string) *InfraLogger {
	return &InfraLogger{l.With().Str(EMAIL, email).Logger()}
}

// InfraAuditEmail is a logging decorator InfraCtxLogger intended to be used for auditing related logs.
func (l *InfraCtxLogger) InfraAuditEmail(email string) *InfraCtxLogger {
	return &InfraCtxLogger{l.With().Str(EMAIL, email).Logger()}
}

// InfraAuditOperation is a logging decorator for InfraLogger intended to be used for auditing related logs.
func (l *InfraLogger) InfraAuditOperation(op string) *InfraLogger {
	return &InfraLogger{l.With().Str(OPERATION, op).Logger()}
}

// InfraAuditOperation is a logging decorator InfraCtxLogger intended to be used for auditing related logs.
func (l *InfraCtxLogger) InfraAuditOperation(op string) *InfraCtxLogger {
	return &InfraCtxLogger{l.With().Str(OPERATION, op).Logger()}
}

// InfraAuditPath is a logging decorator for InfraLogger intended to be used for auditing related logs.
func (l *InfraLogger) InfraAuditPath(path string) *InfraLogger {
	return &InfraLogger{l.With().Str(PATH, path).Logger()}
}

// InfraAuditPath is a logging decorator InfraCtxLogger intended to be used for auditing related logs.
func (l *InfraCtxLogger) InfraAuditPath(path string) *InfraCtxLogger {
	return &InfraCtxLogger{l.With().Str(PATH, path).Logger()}
}

// InfraAuditRequest is a logging decorator for InfraLogger intended to be used for auditing related logs.
func (l *InfraLogger) InfraAuditRequest(req interface{}) *InfraLogger {
	return &InfraLogger{l.With().Str(REQUEST, fmt.Sprintf("%v", req)).Logger()}
}

// InfraAuditRequest is a logging decorator InfraCtxLogger intended to be used for auditing related logs.
func (l *InfraCtxLogger) InfraAuditRequest(req interface{}) *InfraCtxLogger {
	return &InfraCtxLogger{l.With().Str(REQUEST, fmt.Sprintf("%v", req)).Logger()}
}

// InfraAuditResponse is a logging decorator for InfraLogger intended to be used for auditing related logs.
func (l *InfraLogger) InfraAuditResponse(resp interface{}) *InfraLogger {
	return &InfraLogger{l.With().Str(RESPONSE, fmt.Sprintf("%v", resp)).Logger()}
}

// InfraAuditResponse is a logging decorator InfraCtxLogger intended to be used for auditing related logs.
func (l *InfraCtxLogger) InfraAuditResponse(resp interface{}) *InfraCtxLogger {
	return &InfraCtxLogger{l.With().Str(RESPONSE, fmt.Sprintf("%v", resp)).Logger()}
}

// InfraAuditError is a logging decorator InfraCtxLogger intended to be used for auditing related logs.
func (l *InfraLogger) InfraAuditError(err error) *InfraLogger {
	return &InfraLogger{l.With().AnErr(ERROR, err).Logger()}
}

// InfraAuditError is a logging decorator InfraCtxLogger intended to be used for auditing related logs.
func (l *InfraCtxLogger) InfraAuditError(err error) *InfraCtxLogger {
	return &InfraCtxLogger{l.With().AnErr(ERROR, err).Logger()}
}

// InfraAuditStatus is a logging decorator InfraCtxLogger intended to be used for auditing related logs.
func (l *InfraLogger) InfraAuditStatus(status string) *InfraLogger {
	return &InfraLogger{l.With().Str(STATUS, status).Logger()}
}

// InfraAuditStatus is a logging decorator InfraCtxLogger intended to be used for auditing related logs.
func (l *InfraCtxLogger) InfraAuditStatus(status string) *InfraCtxLogger {
	return &InfraCtxLogger{l.With().Str(STATUS, status).Logger()}
}

// InfraErr is an extension for InfraLogger intended to be used for error logging.
func (l *InfraLogger) InfraErr(err error) *zerolog.Event {
	miLogger := &InfraLogger{l.With().Err(err).Logger()}
	return miLogger.Error()
}

// InfraErr is an extension for InfraCtxLogger intended to be used for error logging.
func (l *InfraCtxLogger) InfraErr(err error) *zerolog.Event {
	miLogger := &InfraCtxLogger{l.With().Err(err).Logger()}
	return miLogger.Error()
}

// InfraError is an extension for InfraLogger intended to be used for logging of inline errors.
func (l *InfraLogger) InfraError(format string, args ...interface{}) *zerolog.Event {
	logger := &InfraLogger{l.With().Str("error", fmt.Sprintf(format, args...)).Logger()}
	return logger.Error()
}

// InfraError is an extension for InfraCtxLogger intended to be used for logging of inline errors.
func (l *InfraCtxLogger) InfraError(format string, args ...interface{}) *zerolog.Event {
	logger := &InfraCtxLogger{l.With().Str("error", fmt.Sprintf(format, args...)).Logger()}
	return logger.Error()
}

// CustomWriter captures log messages in a buffer.
type CustomWriter struct {
	Buf *bytes.Buffer
}

func (w *CustomWriter) Write(p []byte) (n int, err error) {
	return w.Buf.Write(p)
}
