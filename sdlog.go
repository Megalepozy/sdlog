package sdlog

import (
	"fmt"
	"github.com/blendle/zapdriver"
	"github.com/google/uuid"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"runtime"
)

type SDLogger interface {
	Lbl(k string, v interface{}) *SDLog
	AddLogTracingID(id string) *SDLog
	Info(message string)
	Error(message string) string
}

type SDLog struct {
	fields []zap.Field
}

func New() *SDLog {
	return &SDLog{}
}

func (s *SDLog) Lbl(k string, v interface{}) *SDLog {
	vs := cast.ToString(v)
	if vs == "" {
		vs = fmt.Sprintf("%#v", v)
	}

	s.fields = append(s.fields, zapdriver.Label(k, vs))

	return s
}

func (s *SDLog) AddLogTracingID(id string) *SDLog {
	s.Lbl("logTracingID", id)
	return s
}

func (s *SDLog) Info(message string) {
	logger := createLogger("stdout")
	defer logger.Sync()

	s.appendSourceLocation()

	logger.Info(message, s.fields...)
}

func (s *SDLog) Error(message string) string {
	logger := createLogger("stderr")
	defer logger.Sync()

	s.appendSourceLocation()

	logTracingID := uuid.New().String()
	s.AddLogTracingID(logTracingID)

	logger.Error(message, s.fields...)

	return logTracingID
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
	s.fields = append(s.fields, zapdriver.SourceLocation(runtime.Caller(2)))
}
