package dispatch

// Filter is a predicate over a typed payload (e.g. *api.Message). Filters
// compose via And/Or/Not for multi-condition matching.
//
// Example:
//
//	f := message.HasPhoto().And(message.InChat(-100123456789))
type Filter[T any] func(payload T) bool

// And returns a Filter that matches iff f and every one of others matches.
func (f Filter[T]) And(others ...Filter[T]) Filter[T] {
	return func(payload T) bool {
		if !f(payload) {
			return false
		}
		for _, o := range others {
			if !o(payload) {
				return false
			}
		}
		return true
	}
}

// Or returns a Filter that matches iff f matches OR any of others matches.
func (f Filter[T]) Or(others ...Filter[T]) Filter[T] {
	return func(payload T) bool {
		if f(payload) {
			return true
		}
		for _, o := range others {
			if o(payload) {
				return true
			}
		}
		return false
	}
}

// Not returns a Filter that inverts f.
func (f Filter[T]) Not() Filter[T] {
	return func(payload T) bool { return !f(payload) }
}

// All combines filters with AND. Returns a Filter that matches when all match.
// Returns a filter that always matches when filters is empty.
func All[T any](filters ...Filter[T]) Filter[T] {
	return func(payload T) bool {
		for _, f := range filters {
			if !f(payload) {
				return false
			}
		}
		return true
	}
}

// Any combines filters with OR. Returns a Filter that matches when at least
// one matches. Returns a filter that never matches when filters is empty.
func Any[T any](filters ...Filter[T]) Filter[T] {
	return func(payload T) bool {
		for _, f := range filters {
			if f(payload) {
				return true
			}
		}
		return false
	}
}
