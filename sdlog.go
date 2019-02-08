package sdlog

import (
	"fmt"
	"runtime"

	"github.com/blendle/zapdriver"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"go.uber.org/zap"
)

type SDLog struct {
	callerSkip int
	fields     []zap.Field
}

func New(callerSkip ...int) *SDLog {
	skip := 2
	if len(callerSkip) != 0 {
		skip = callerSkip[0]
	}

	return &SDLog{callerSkip: skip}
}

func (s *SDLog) Info(message string, options ...func(s *SDLog)) {
	logger := createLogger("stdout")
	defer logger.Sync()

	s.appendSourceLocation()

	for _, option := range options {
		option(s)
	}

	logger.Info(message, s.fields...)
}

func (s *SDLog) Error(message string, options ...func(s *SDLog)) string {
	logger := createLogger("stderr")
	defer logger.Sync()

	s.appendSourceLocation()

	logTracingID := uuid.New().String()
	options = append(options, s.AddLogTracingID(logTracingID))

	for _, option := range options {
		option(s)
	}

	logger.Error(message, s.fields...)

	return logTracingID
}

func (s *SDLog) Lbl(k string, v interface{}) func(*SDLog) {
	return func(s *SDLog) {
		vs := cast.ToString(v)
		if vs == "" {
			vs = fmt.Sprintf("%#v", v)
		}

		s.fields = append(s.fields, zapdriver.Label(k, vs))
	}
}

func (s *SDLog) AddLogTracingID(id string) func(*SDLog) {
	return s.Lbl("logTracingID", id)
}

func createLogger(outputStream string) *zap.Logger {
	config := zapdriver.NewProductionConfig()
	config.OutputPaths = []string{outputStream}
	logger, err := config.Build(zapdriver.WrapCore())
	if err != nil {
		panic(fmt.Sprintf("Unexpected error while building logger: %v", err))
	}

	return logger
}

func (s *SDLog) appendSourceLocation() {
	s.fields = append(s.fields, zapdriver.SourceLocation(runtime.Caller(s.callerSkip)))
}
