package api

import (
	"testing"
)

func TestSenderID(t *testing.T) {
	tests := []struct {
		name   string
		sender *Sender
		want   int64
	}{
		{
			name:   "nil sender",
			sender: nil,
			want:   0,
		},
		{
			name:   "empty sender",
			sender: &Sender{},
			want:   0,
		},
		{
			name: "user only",
			sender: &Sender{
				User: &User{ID: 123},
			},
			want: 123,
		},
		{
			name: "chat only",
			sender: &Sender{
				Chat: &Chat{ID: 456},
			},
			want: 456,
		},
		{
			name: "chat prefers over user",
			sender: &Sender{
				User: &User{ID: 123},
				Chat: &Chat{ID: 456},
			},
			want: 456,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sender.ID()
			if got != tt.want {
				t.Errorf("ID() = %d, want %d", got, tt.want)
			}
		})
	}
}

func chatEqual(a, b *Chat) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.ID == b.ID
}

func userEqual(a, b *User) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.ID == b.ID
}

func TestSenderIsAnonymousAdmin(t *testing.T) {
	tests := []struct {
		name   string
		sender *Sender
		want   bool
	}{
		{
			name:   "nil sender",
			sender: nil,
			want:   false,
		},
		{
			name:   "no chat",
			sender: &Sender{User: &User{ID: 123}, ChatID: 456},
			want:   false,
		},
		{
			name: "chat id matches (anonymous admin)",
			sender: &Sender{
				Chat:   &Chat{ID: 789},
				ChatID: 789,
			},
			want: true,
		},
		{
			name: "chat id differs (not anonymous admin)",
			sender: &Sender{
				Chat:   &Chat{ID: 789},
				ChatID: 456,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sender.IsAnonymousAdmin()
			if got != tt.want {
				t.Errorf("IsAnonymousAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSenderIsAnonymousChannel(t *testing.T) {
	tests := []struct {
		name   string
		sender *Sender
		want   bool
	}{
		{
			name:   "nil sender",
			sender: nil,
			want:   false,
		},
		{
			name:   "no chat",
			sender: &Sender{User: &User{ID: 123}, ChatID: 456},
			want:   false,
		},
		{
			name: "chat id differs (anonymous channel)",
			sender: &Sender{
				Chat:   &Chat{ID: 789},
				ChatID: 456,
			},
			want: true,
		},
		{
			name: "chat id matches (not anonymous channel)",
			sender: &Sender{
				Chat:   &Chat{ID: 789},
				ChatID: 789,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.sender.IsAnonymousChannel()
			if got != tt.want {
				t.Errorf("IsAnonymousChannel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageGetSender(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
		want *Sender
	}{
		{
			name: "nil message",
			msg:  nil,
			want: &Sender{},
		},
		{
			name: "regular user message",
			msg: &Message{
				From: &User{ID: 123},
				Chat: Chat{ID: 456},
			},
			want: &Sender{
				User:   &User{ID: 123},
				ChatID: 456,
			},
		},
		{
			name: "channel forward",
			msg: &Message{
				From:       &User{ID: 123},
				SenderChat: &Chat{ID: 789},
				Chat:       Chat{ID: 456},
			},
			want: &Sender{
				User:   &User{ID: 123},
				Chat:   &Chat{ID: 789},
				ChatID: 456,
			},
		},
		{
			name: "anonymous admin",
			msg: &Message{
				SenderChat:      &Chat{ID: 456},
				Chat:            Chat{ID: 456},
				AuthorSignature: "Admin Signature",
			},
			want: &Sender{
				Chat:            &Chat{ID: 456},
				ChatID:          456,
				AuthorSignature: "Admin Signature",
			},
		},
		{
			name: "anonymous channel post",
			msg: &Message{
				SenderChat: &Chat{ID: 789},
				Chat:       Chat{ID: 456},
			},
			want: &Sender{
				Chat:   &Chat{ID: 789},
				ChatID: 456,
			},
		},
		{
			name: "automatic forward",
			msg: &Message{
				From: &User{ID: 123},
				IsAutomaticForward: func() *bool {
					b := true
					return &b
				}(),
				Chat: Chat{ID: 456},
			},
			want: &Sender{
				User:               &User{ID: 123},
				IsAutomaticForward: true,
				ChatID:             456,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.msg.GetSender()
			if got == nil {
				t.Fatal("GetSender() returned nil")
			}
			if !userEqual(got.User, tt.want.User) {
				t.Errorf("User: got %v, want %v", got.User, tt.want.User)
			}
			if !chatEqual(got.Chat, tt.want.Chat) {
				t.Errorf("Chat: got %v, want %v", got.Chat, tt.want.Chat)
			}
			if got.IsAutomaticForward != tt.want.IsAutomaticForward {
				t.Errorf("IsAutomaticForward: got %v, want %v", got.IsAutomaticForward, tt.want.IsAutomaticForward)
			}
			if got.ChatID != tt.want.ChatID {
				t.Errorf("ChatID: got %d, want %d", got.ChatID, tt.want.ChatID)
			}
			if got.AuthorSignature != tt.want.AuthorSignature {
				t.Errorf("AuthorSignature: got %q, want %q", got.AuthorSignature, tt.want.AuthorSignature)
			}
		})
	}
}

func TestMessageReactionUpdatedGetSender(t *testing.T) {
	tests := []struct {
		name string
		mru  *MessageReactionUpdated
		want *Sender
	}{
		{
			name: "nil reaction",
			mru:  nil,
			want: &Sender{},
		},
		{
			name: "user reaction",
			mru: &MessageReactionUpdated{
				User: &User{ID: 123},
				Chat: Chat{ID: 456},
			},
			want: &Sender{
				User:   &User{ID: 123},
				ChatID: 456,
			},
		},
		{
			name: "anonymous reaction",
			mru: &MessageReactionUpdated{
				ActorChat: &Chat{ID: 789},
				Chat:      Chat{ID: 456},
			},
			want: &Sender{
				Chat:   &Chat{ID: 789},
				ChatID: 456,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.mru.GetSender()
			if got == nil {
				t.Fatal("GetSender() returned nil")
			}
			if !userEqual(got.User, tt.want.User) {
				t.Errorf("User: got %v, want %v", got.User, tt.want.User)
			}
			if !chatEqual(got.Chat, tt.want.Chat) {
				t.Errorf("Chat: got %v, want %v", got.Chat, tt.want.Chat)
			}
			if got.ChatID != tt.want.ChatID {
				t.Errorf("ChatID: got %d, want %d", got.ChatID, tt.want.ChatID)
			}
		})
	}
}

func TestPollAnswerGetSender(t *testing.T) {
	tests := []struct {
		name string
		pa   *PollAnswer
		want *Sender
	}{
		{
			name: "nil poll answer",
			pa:   nil,
			want: &Sender{},
		},
		{
			name: "user vote",
			pa: &PollAnswer{
				User: &User{ID: 123},
			},
			want: &Sender{
				User: &User{ID: 123},
			},
		},
		{
			name: "anonymous vote",
			pa: &PollAnswer{
				VoterChat: &Chat{ID: 789},
			},
			want: &Sender{
				Chat: &Chat{ID: 789},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.pa.GetSender()
			if got == nil {
				t.Fatal("GetSender() returned nil")
			}
			if !userEqual(got.User, tt.want.User) {
				t.Errorf("User: got %v, want %v", got.User, tt.want.User)
			}
			if !chatEqual(got.Chat, tt.want.Chat) {
				t.Errorf("Chat: got %v, want %v", got.Chat, tt.want.Chat)
			}
		})
	}
}
