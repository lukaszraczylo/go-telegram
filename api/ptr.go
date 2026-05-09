package api

// Ptr returns a pointer to v. Useful for optional scalar fields where
// the wire format must distinguish absent (nil) from an explicit zero
// value (e.g. DisableNotification: api.Ptr(false) to override the
// chat default).
//
// For untyped literals, supply the type parameter explicitly:
//
//	Limit: api.Ptr[int64](5)
//
// For already-typed values, type inference handles it:
//
//	var n int64 = 5
//	Limit: api.Ptr(n)
func Ptr[T any](v T) *T { return &v }
