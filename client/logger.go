package client

// Logger is a slog-shaped logging interface. Users pass any compatible
// implementation via WithLogger. The default is NoopLogger, which discards
// everything.
type Logger interface {
	Debug(msg string, attrs ...any)
	Info(msg string, attrs ...any)
	Warn(msg string, attrs ...any)
	Error(msg string, attrs ...any)
}

// NoopLogger discards all log records. It is the zero-value safe default.
type NoopLogger struct{}

func (NoopLogger) Debug(string, ...any) {}
func (NoopLogger) Info(string, ...any)  {}
func (NoopLogger) Warn(string, ...any)  {}
func (NoopLogger) Error(string, ...any) {}
