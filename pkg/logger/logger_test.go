package logger

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestInfo(t *testing.T) {
	logs := &logSink{}
	logger := ZeroLogger(zerolog.New(logs))
	logger.Info("test")

	assert.Equal(t, 1, len(logs.logs))
	assert.Equal(t, `{"level":"info","message":"test"}` + "\n", logs.logs[0])
}
