package client

// Result is the universal Telegram API response envelope. Every successful
// response is shaped {"ok":true,"result":T,...}; failure responses set ok
// to false and populate ErrorCode / Description / Parameters.
//
// Result is generic over T so generated method wrappers can decode the
// strongly-typed payload directly. Users do not normally construct or
// inspect Result values; method wrappers unwrap them and return either
// the typed payload or a *APIError.
type Result[T any] struct {
	OK          bool                `json:"ok"`
	Result      T                   `json:"result,omitempty"`
	ErrorCode   int                 `json:"error_code,omitempty"`
	Description string              `json:"description,omitempty"`
	Parameters  *ResponseParameters `json:"parameters,omitempty"`
}

// ResponseParameters is the optional metadata Telegram includes on certain
// failures. The most common is RetryAfter (seconds) on 429 responses.
//
// This type is duplicated in package api for users; keeping a copy here
// avoids an import cycle (api imports client, not vice versa).
type ResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id,omitempty"`
	RetryAfter      int   `json:"retry_after,omitempty"`
}
