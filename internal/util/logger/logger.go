package logger

import (
	"context"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

type Log interface {
	Debug(context.Context, ...interface{})
	Info(context.Context, ...interface{})
	Error(context.Context, ...interface{})
	Warning(context.Context, ...interface{})
	WithError(context.Context, error) context.Context
	WithField(ctx context.Context, k string, v interface{}) context.Context
	WithFields(context.Context, map[string]interface{}) context.Context
}

type ctxKey struct{}

type Logger struct {
	z      *zap.SugaredLogger
	sentry bool
}

func New() (*Logger, error) {
	z, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &Logger{z: z.Sugar()}, nil
}

// EnableSentry включает отправку Error и Warning в Sentry; пустой dsn оставляет её выключенной.
func (l *Logger) EnableSentry(dsn, env string) error {
	if dsn == "" {
		return nil
	}
	if err := sentry.Init(sentry.ClientOptions{Dsn: dsn, Environment: env, AttachStacktrace: true}); err != nil {
		return fmt.Errorf("failed to init sentry: %w", err)
	}
	l.sentry = true
	return nil
}

func (l *Logger) Flush() {
	_ = l.z.Sync()
	if l.sentry {
		sentry.Flush(2 * time.Second)
	}
}

func (l *Logger) Debug(ctx context.Context, args ...interface{}) {
	msg, kv := split(args)
	l.z.With(extractCtxFields(ctx)...).Debugw(msg, kv...)
}

func (l *Logger) Info(ctx context.Context, args ...interface{}) {
	msg, kv := split(args)
	l.z.With(extractCtxFields(ctx)...).Infow(msg, kv...)
}

func (l *Logger) Error(ctx context.Context, args ...interface{}) {
	msg, kv := split(args)
	l.z.With(extractCtxFields(ctx)...).Errorw(msg, kv...)
	l.sendToSentry(ctx, sentry.LevelError, msg, kv, args)
}

func (l *Logger) Warning(ctx context.Context, args ...interface{}) {
	msg, kv := split(args)
	l.z.With(extractCtxFields(ctx)...).Warnw(msg, kv...)
	l.sendToSentry(ctx, sentry.LevelWarning, msg, kv, args)
}

func (l *Logger) WithError(ctx context.Context, err error) context.Context {
	return l.WithField(ctx, "error", err.Error())
}

func (l *Logger) WithField(ctx context.Context, k string, v interface{}) context.Context {
	existing := extractCtxFields(ctx)
	f := make([]interface{}, len(existing)+2)
	copy(f, existing)
	f[len(existing)] = k
	f[len(existing)+1] = v
	return context.WithValue(ctx, ctxKey{}, f)
}

func (l *Logger) WithFields(ctx context.Context, fields map[string]interface{}) context.Context {
	existing := extractCtxFields(ctx)
	f := make([]interface{}, len(existing), len(existing)+len(fields)*2)
	copy(f, existing)
	for k, v := range fields {
		f = append(f, k, v)
	}
	return context.WithValue(ctx, ctxKey{}, f)
}

// sendToSentry шлёт error из аргументов как исключение со стектрейсом, а без него — как сообщение.
func (l *Logger) sendToSentry(ctx context.Context, level sentry.Level, msg string, kv, args []interface{}) {
	if !l.sentry {
		return
	}
	// Клон хаба, чтобы параллельные запросы не смешивали контекст в общем scope.
	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)
		fields := append(append([]interface{}{}, extractCtxFields(ctx)...), kv...)
		data := sentry.Context{"message": msg}
		for i := 0; i+1 < len(fields); i += 2 {
			data[fmt.Sprint(fields[i])] = fields[i+1]
		}
		scope.SetContext("log", data)
	})
	for _, a := range args {
		if err, ok := a.(error); ok && err != nil {
			hub.CaptureException(err)
			return
		}
	}
	hub.CaptureMessage(msg)
}

// split делит аргументы вызова на сообщение и поля: первый аргумент — сообщение, дальше пары «ключ, значение»;
// error в любом месте становится полем error, значение без ключа — полем arg.
func split(args []interface{}) (string, []interface{}) {
	if len(args) == 0 {
		return "", nil
	}
	msg := fmt.Sprint(args[0])
	rest := args[1:]
	var kv []interface{}
	for i := 0; i < len(rest); i++ {
		if err, ok := rest[i].(error); ok && err != nil {
			// fmt.Sprint, а не err.Error(): ошибка-указатель nil внутри интерфейса не роняет процесс.
			kv = append(kv, "error", fmt.Sprint(err))
			continue
		}
		if key, ok := rest[i].(string); ok && i+1 < len(rest) {
			kv = append(kv, key, rest[i+1])
			i++
			continue
		}
		kv = append(kv, "arg", rest[i])
	}
	return msg, kv
}

func extractCtxFields(ctx context.Context) []interface{} {
	if f, ok := ctx.Value(ctxKey{}).([]interface{}); ok {
		return f
	}
	return nil
}
