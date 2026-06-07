package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	LogLevel  string = "info"
	LogFormat string = "console"
)

type SelfLogger struct {
	*zap.Logger
}

var Logger *SelfLogger

var logLevelMap = map[string]zapcore.Level{
	"debug": zapcore.DebugLevel,
	"info":  zapcore.InfoLevel,
	"warn":  zapcore.WarnLevel,
	"error": zapcore.ErrorLevel,
}

var logFormatMap = map[string]zapcore.EncoderConfig{
	"console": {
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "msg",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.RFC3339TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	},
	"json": {
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		MessageKey:     "msg",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.EpochMillisTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	},
}

func (l *SelfLogger) Debugf(template string, args ...any) {
	l.Logger.Debug(fmt.Sprintf(template, args...))
}

func (l *SelfLogger) Infof(template string, args ...any) {
	l.Logger.Info(fmt.Sprintf(template, args...))
}

func (l *SelfLogger) Errorf(template string, args ...any) {
	l.Logger.Error(fmt.Sprintf(template, args...))
}

func (l *SelfLogger) Warnf(template string, args ...any) {
	l.Logger.Warn(fmt.Sprintf(template, args...))
}

func (l *SelfLogger) Panicf(template string, args ...any) {
	l.Logger.Panic(fmt.Sprintf(template, args...))
}

func init() {
	// cobra.OnInitialize(initLogger)
	initLogger()
}

func initLogger() {
	level, ok := logLevelMap[strings.ToLower(LogLevel)]
	if !ok {
		fmt.Printf("unsupported log level: %s\n", LogLevel)
		os.Exit(1)
	}
	enc, ok := logFormatMap[strings.ToLower(LogFormat)]
	if !ok {
		fmt.Printf("unsupported log format: %s\n", LogFormat)
		os.Exit(1)
	}
	c := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		DisableCaller:     true,
		DisableStacktrace: true,
		Encoding:          strings.ToLower(LogFormat),
		EncoderConfig:     enc,
		OutputPaths:       []string{"stderr"},
		ErrorOutputPaths:  []string{"stderr"},
	}
	var err error
	logger, err := c.Build()
	Logger = &SelfLogger{Logger: logger}
	if err != nil {
		fmt.Printf("failed to initialize logger: %s\n", err)
		os.Exit(1)
	}
}
