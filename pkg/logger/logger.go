package logger

import (
	"fmt"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type (
	Logger interface {
		Info(msg string, keysAndValues ...interface{})
		Debug(msg string, keysAndValues ...interface{})
		Warn(msg string, keysAndValues ...interface{})
		Error(msg string, keysAndValues ...interface{})
	}

	zerologger struct{
		logger zerolog.Logger
	}
)

func NewZeroLogger() Logger {
	return ZeroLogger(log.Logger)
}

func ZeroLogger(log zerolog.Logger) Logger {
	return zerologger{logger: log}
}

func (z zerologger) Info(msg string, keysAndValues ...interface{}) {
	e := z.logger.Info()
	for i := 0; i < len(keysAndValues); i += 2 {
		e.Interface(fmt.Sprintf("%s", keysAndValues[i]), keysAndValues[i+1])
	}
	e.Msg(msg)
}

func (z zerologger) Debug(msg string, keysAndValues ...interface{}) {
	e := z.logger.Debug()
	for i := 0; i < len(keysAndValues); i += 2 {
		e.Interface(fmt.Sprintf("%s", keysAndValues[i]), keysAndValues[i+1])
	}
	e.Msg(msg)
}

func (z zerologger) Warn(msg string, keysAndValues ...interface{}) {
	e := z.logger.Warn()
	for i := 0; i < len(keysAndValues); i += 2 {
		e.Interface(fmt.Sprintf("%s", keysAndValues[i]), keysAndValues[i+1])
	}
	e.Msg(msg)
}

func (z zerologger) Error(msg string, keysAndValues ...interface{}) {
	e := z.logger.Error()
	for i := 0; i < len(keysAndValues); i += 2 {
		e.Interface(fmt.Sprintf("%s", keysAndValues[i]), keysAndValues[i+1])
	}
	e.Msg(msg)
}

// test helper

type logSink struct{ logs []string }

func (l *logSink) Index(i int) string { return l.logs[i] }

func (l *logSink) Write(p []byte) (n int, err error) {
	l.logs = append(l.logs, string(p))
	return len(p), nil
}
