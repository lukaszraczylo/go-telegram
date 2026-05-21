package client

// Version is the released version of the go-telegram library. Keep in sync
// with the most recent git tag; this constant is bumped manually on each
// release. Exposed as a var (not const) so downstream applications may
// override it via linker flags:
//
//	go build -ldflags="-X github.com/lukaszraczylo/go-telegram/client.Version=1.2.3"
//
// The value is also forwarded as the version field of the anonymous usage
// ping that fires on the first call to New (see fireTelemetryOnce).
var Version = "0.7.11"
