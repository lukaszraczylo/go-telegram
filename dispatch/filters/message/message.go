// Package message provides Filter helpers for *api.Message payloads.
package message

import (
	"regexp"
	"strings"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/dispatch"
)

// Text returns a Filter that matches messages whose Text matches pattern (regex).
// Panics at registration time on an invalid pattern.
func Text(pattern string) dispatch.Filter[*api.Message] {
	re := regexp.MustCompile(pattern)
	return func(m *api.Message) bool {
		return m != nil && re.MatchString(m.Text)
	}
}

// TextEquals returns a Filter that matches messages whose Text equals s exactly.
func TextEquals(s string) dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		return m != nil && m.Text == s
	}
}

// TextPrefix returns a Filter that matches messages whose Text starts with prefix.
func TextPrefix(prefix string) dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		return m != nil && strings.HasPrefix(m.Text, prefix)
	}
}

// TextContains returns a Filter that matches messages whose Text contains sub.
func TextContains(sub string) dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		return m != nil && strings.Contains(m.Text, sub)
	}
}

// Command returns a Filter that matches messages whose first entity is a
// bot_command equal to "/<name>" (with or without "@BotName" suffix).
func Command(name string) dispatch.Filter[*api.Message] {
	want := "/" + strings.TrimPrefix(name, "/")
	return func(m *api.Message) bool {
		if m == nil || len(m.Entities) == 0 || m.Text == "" {
			return false
		}
		first := m.Entities[0]
		if first.Type != string(api.EntityBotCommand) || first.Offset != 0 {
			return false
		}
		end := int(first.Length)
		runes := []rune(m.Text)
		if end > len(runes) {
			return false
		}
		cmd := string(runes[:end])
		if i := strings.Index(cmd, "@"); i >= 0 {
			cmd = cmd[:i]
		}
		return cmd == want
	}
}

// AnyCommand returns a Filter that matches any message starting with a
// bot_command entity at offset 0.
func AnyCommand() dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		if m == nil || len(m.Entities) == 0 {
			return false
		}
		first := m.Entities[0]
		return first.Type == string(api.EntityBotCommand) && first.Offset == 0
	}
}

// IsReply returns a Filter that matches messages that have ReplyToMessage set.
func IsReply() dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		return m != nil && m.ReplyToMessage != nil
	}
}

// IsForward returns a Filter that matches messages that have ForwardOrigin set.
func IsForward() dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		return m != nil && m.ForwardOrigin != nil
	}
}

// HasPhoto returns a Filter that matches messages with a Photo attachment.
func HasPhoto() dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		return m != nil && len(m.Photo) > 0
	}
}

// HasDocument returns a Filter that matches messages with a Document attachment.
func HasDocument() dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		return m != nil && m.Document != nil
	}
}

// HasEntity returns a Filter that matches messages whose Entities contain at
// least one entity of type t (e.g. string(api.EntityBotCommand)).
func HasEntity(t string) dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		if m == nil {
			return false
		}
		for _, e := range m.Entities {
			if e.Type == t {
				return true
			}
		}
		return false
	}
}

// ChatType returns a Filter that matches messages whose Chat.Type equals t.
func ChatType(t api.ChatType) dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		return m != nil && m.Chat.Type == string(t)
	}
}

// FromUser returns a Filter that matches messages whose From.ID equals userID.
func FromUser(userID int64) dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		return m != nil && m.From != nil && m.From.ID == userID
	}
}

// InChat returns a Filter that matches messages whose Chat.ID equals chatID.
func InChat(chatID int64) dispatch.Filter[*api.Message] {
	return func(m *api.Message) bool {
		return m != nil && m.Chat.ID == chatID
	}
}
