// Package conversation implements a stateful conversation handler for the
// go-telegram dispatch router. It provides a state-machine abstraction over
// multi-step Telegram bot interactions, with pluggable storage and flexible
// key strategies.
package conversation

// State is a label identifying a node in the conversation graph.
// The empty string is the implicit "no active conversation" state.
type State string
