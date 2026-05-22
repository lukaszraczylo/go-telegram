package client

// Version is a fallback version string used only when Go's build info is
// unavailable (replace directives, detached `go run`) or has been overridden
// via linker flags. The authoritative version forwarded to telemetry is
// resolved at runtime by [telemetry.SendForModule] from the build info of
// whatever binary linked this library, so this constant does NOT need to be
// bumped on every release. Exposed as a var (not const) for ldflag override:
//
//	go build -ldflags="-X github.com/lukaszraczylo/go-telegram/client.Version=1.2.3"
var Version = "0.0.0-fallback"
