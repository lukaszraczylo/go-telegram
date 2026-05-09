package api

// Sender condenses the various ways a Telegram update can identify the
// originator of a message or reaction into a single shape. Use the
// GetSender methods on supported types to construct one.
type Sender struct {
	// User is the human user who sent the update, when applicable.
	User *User
	// Chat is the chat that sent the update (channel forwards,
	// anonymous group admins, anonymous channel posts).
	Chat *Chat
	// IsAutomaticForward is true when the update originated as an
	// automatic forward from a linked channel.
	IsAutomaticForward bool
	// ChatID is the chat the update was delivered into. Used to
	// distinguish "this user" from "this anonymous admin posting
	// in <chat>" when User is nil.
	ChatID int64
	// AuthorSignature is the custom title of an anonymous group
	// administrator. Only meaningful when Chat == this chat.
	AuthorSignature string
}

// ID returns the most-specific identifier available: prefers Chat.ID
// over User.ID. Returns 0 if neither is set.
func (s *Sender) ID() int64 {
	if s == nil {
		return 0
	}
	if s.Chat != nil {
		return s.Chat.ID
	}
	if s.User != nil {
		return s.User.ID
	}
	return 0
}

// IsAnonymousAdmin reports whether the sender is a group admin posting
// anonymously (Chat equals the message's own chat).
func (s *Sender) IsAnonymousAdmin() bool {
	return s != nil && s.Chat != nil && s.Chat.ID == s.ChatID
}

// IsAnonymousChannel reports whether the sender is an anonymous
// channel post (Chat differs from the message's own chat).
func (s *Sender) IsAnonymousChannel() bool {
	return s != nil && s.Chat != nil && s.Chat.ID != s.ChatID
}

// GetSender constructs a Sender for a Message. The result is never nil.
func (m *Message) GetSender() *Sender {
	if m == nil {
		return &Sender{}
	}
	isAuto := false
	if m.IsAutomaticForward != nil {
		isAuto = *m.IsAutomaticForward
	}
	return &Sender{
		User:               m.From,
		Chat:               m.SenderChat,
		IsAutomaticForward: isAuto,
		ChatID:             m.Chat.ID,
		AuthorSignature:    m.AuthorSignature,
	}
}

// GetSender constructs a Sender for a MessageReactionUpdated.
func (mru *MessageReactionUpdated) GetSender() *Sender {
	if mru == nil {
		return &Sender{}
	}
	return &Sender{
		User:   mru.User,
		Chat:   mru.ActorChat,
		ChatID: mru.Chat.ID,
	}
}

// GetSender constructs a Sender for a PollAnswer.
func (pa *PollAnswer) GetSender() *Sender {
	if pa == nil {
		return &Sender{}
	}
	return &Sender{
		User: pa.User,
		Chat: pa.VoterChat,
	}
}
