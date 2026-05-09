package client

import "testing"

func TestNoopLogger_DoesNotPanic(t *testing.T) {
	var l Logger = NoopLogger{}
	l.Debug("d", "k", "v")
	l.Info("i")
	l.Warn("w")
	l.Error("e")
}
