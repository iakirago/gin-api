package logger

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	"github.com/why444216978/go-util/conversion"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config is used to parse configuration file
// logger should be controlled with Options
type Config struct {
	InfoFile  string
	ErrorFile string
	Level     string
}

type Logger struct {
	*zap.Logger
	opts  *Options
	level zapcore.Level
}

type Options struct {
	level       string
	callSkip    int
	module      string
	serviceName string
	infoWriter  io.Writer
	errorWriter io.Writer
}

type Option func(l *Options)

func defaultOptions() *Options {
	return &Options{
		level:       "info",
		callSkip:    1,
		module:      "default",
		serviceName: "default",
		infoWriter:  os.Stdout,
		errorWriter: os.Stdout,
	}
}

func WithCallerSkip(skip int) Option {
	return func(o *Options) { o.callSkip = skip }
}

func WithModule(module string) Option {
	return func(o *Options) { o.module = module }
}

func WithServiceName(serviceName string) Option {
	return func(o *Options) { o.serviceName = serviceName }
}

func WithInfoWriter(w io.Writer) Option {
	return func(o *Options) { o.infoWriter = w }
}

func WithErrorWriter(w io.Writer) Option {
	return func(o *Options) { o.errorWriter = w }
}

func WithLevel(l string) Option {
	return func(o *Options) { o.level = l }
}

func NewLogger(cfg *Config, options ...Option) (l *Logger, err error) {
	opts := defaultOptions()
	for _, o := range options {
		o(opts)
	}

	level, err := zapLevel(opts.level)
	if err != nil {
		return
	}

	l = &Logger{
		level: level,
		opts:  opts,
	}

	encoder := l.formatEncoder()

	infoEnabler := l.infoEnabler()
	errorEnabler := l.errorEnabler()

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(opts.infoWriter), infoEnabler),
		zapcore.NewCore(encoder, zapcore.AddSync(opts.errorWriter), errorEnabler),
	)

	fields := []zapcore.Field{
		zap.String(Module, l.opts.module),
		zap.String(SericeName, l.opts.serviceName),
	}

	l.Logger = zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(errorEnabler),
		zap.AddCallerSkip(l.opts.callSkip),
		zap.Fields(fields...),
	)

	return
}

func (l *Logger) infoEnabler() zap.LevelEnablerFunc {
	return zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		if lvl < l.level {
			return false
		}
		return lvl <= zapcore.InfoLevel
	})
}

func (l *Logger) errorEnabler() zap.LevelEnablerFunc {
	return zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		if lvl < l.level {
			return false
		}
		return lvl >= zapcore.WarnLevel
	})
}

func (l *Logger) formatEncoder() zapcore.Encoder {
	return zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		MessageKey:    "msg",
		LevelKey:      "level",
		EncodeLevel:   zapcore.CapitalLevelEncoder,
		TimeKey:       "time",
		CallerKey:     "file",
		FunctionKey:   "func",
		StacktraceKey: "stack",
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		},
		EncodeCaller: zapcore.ShortCallerEncoder,
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		},
	})
}

func (l *Logger) GetLevel() zapcore.Level {
	return l.level
}

func zapLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug", "DEBUG":
		return zapcore.DebugLevel, nil
	case "info", "INFO", "":
		return zapcore.InfoLevel, nil
	case "warn", "WARN":
		return zapcore.WarnLevel, nil
	case "error", "ERROR":
		return zapcore.ErrorLevel, nil
	case "dpanic", "DPANIC":
		return zapcore.DPanicLevel, nil
	case "panic", "PANIC":
		return zapcore.PanicLevel, nil
	case "fatal", "FATAL":
		return zapcore.FatalLevel, nil
	default:
		return 0, errors.New("error level:" + level)
	}
}

func (l *Logger) Debug(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, l.extractFields(ctx, fields...)...)
}

func (l *Logger) Info(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Info(msg, l.extractFields(ctx, fields...)...)
}

func (l *Logger) Warn(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, l.extractFields(ctx, fields...)...)
}

func (l *Logger) Error(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Error(msg, l.extractFields(ctx, fields...)...)
}

func (l *Logger) Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, l.extractFields(ctx, fields...)...)
}

func (l *Logger) extractFields(ctx context.Context, fields ...zap.Field) []zap.Field {
	fieldsMap, _ := conversion.StructToMap(ValueHTTPFields(ctx))
	target := make(map[string]zap.Field, len(fieldsMap))
	for k, v := range fieldsMap {
		target[k] = zap.Reflect(k, v)
	}

	for _, f := range fields {
		target[f.Key] = f
	}

	new := make([]zap.Field, 0)
	for _, f := range target {
		new = append(new, f)
	}

	return new
}
