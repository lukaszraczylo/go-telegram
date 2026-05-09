package client

import "regexp"

// tokenInURL matches a Telegram bot token segment in a URL path. Tokens
// have the form <bot_id>:<api_key>, where bot_id is digits and api_key
// is 35 base64-url characters. The pattern is conservative: matches
// /bot<id>:<key>/ to avoid false positives.
var tokenInURL = regexp.MustCompile(`/bot(\d{5,15}):([A-Za-z0-9_-]{30,40})/`)

// redactToken replaces any bot token in s with /bot<REDACTED>/. Used by
// error formatters so logs don't leak credentials.
func redactToken(s string) string {
	return tokenInURL.ReplaceAllString(s, "/bot<REDACTED>/")
}
