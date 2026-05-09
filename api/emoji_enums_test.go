package api

import (
	"reflect"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

// TestDiceEmoji_Constants pins the canonical six dice-emoji values so a
// regen, refactor, or accidental rename can't silently break the wire
// contract.
func TestDiceEmoji_Constants(t *testing.T) {
	cases := []struct {
		name string
		got  DiceEmoji
		want string
	}{
		{"Dice", DiceEmojiDice, "🎲"},
		{"Dart", DiceEmojiDart, "🎯"},
		{"Basketball", DiceEmojiBasketball, "🏀"},
		{"Football", DiceEmojiFootball, "⚽"},
		{"Bowling", DiceEmojiBowling, "🎳"},
		{"SlotMachine", DiceEmojiSlotMachine, "🎰"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			require.Equal(t, c.want, string(c.got))
		})
	}
}

// TestSendDiceParams_EmojiFieldType asserts the codegen override wired
// SendDiceParams.Emoji to the typed enum (not plain string). Reflection
// catches a regression even if the file compiles via implicit string
// conversion of an untyped literal.
func TestSendDiceParams_EmojiFieldType(t *testing.T) {
	rt := reflect.TypeOf(SendDiceParams{})
	f, ok := rt.FieldByName("Emoji")
	require.True(t, ok, "SendDiceParams.Emoji not present")
	require.Equal(t, "DiceEmoji", f.Type.Name())
}

// TestSendDiceParams_MarshalJSON exercises the marshalled wire form to
// prove the typed enum still serialises as a JSON string holding the
// raw emoji bytes — i.e. the type override doesn't accidentally
// double-encode.
func TestSendDiceParams_MarshalJSON(t *testing.T) {
	p := &SendDiceParams{
		ChatID: ChatIDFromInt(1),
		Emoji:  DiceEmojiBasketball,
	}
	data, err := json.Marshal(p)
	require.NoError(t, err)
	require.Contains(t, string(data), `"emoji":"🏀"`)
}

// TestReactionEmoji_Constants spot-checks a representative slice of the
// 73-value enum. A full enumeration would be redundant — the test is
// here to lock the wire form, not to retest the const-block.
func TestReactionEmoji_Constants(t *testing.T) {
	require.Equal(t, "👍", string(ReactionEmojiThumbsUp))
	require.Equal(t, "👎", string(ReactionEmojiThumbsDown))
	require.Equal(t, "❤", string(ReactionEmojiHeart))
	require.Equal(t, "🔥", string(ReactionEmojiFire))
	require.Equal(t, "💯", string(ReactionEmojiHundredPoints))
	require.Equal(t, "🤡", string(ReactionEmojiClown))
}

// TestReactionTypeEmoji_FieldType asserts the codegen override wired
// ReactionTypeEmoji.Emoji to the typed enum.
func TestReactionTypeEmoji_FieldType(t *testing.T) {
	rt := reflect.TypeOf(ReactionTypeEmoji{})
	f, ok := rt.FieldByName("Emoji")
	require.True(t, ok, "ReactionTypeEmoji.Emoji not present")
	require.Equal(t, "ReactionEmoji", f.Type.Name())
}

// TestReactionTypeEmoji_RoundTrip proves a typed-enum value survives
// JSON marshal → unmarshal cycle without losing fidelity. The
// discriminator MarshalJSON on ReactionTypeEmoji forces type="emoji",
// so we set it explicitly here for symmetry with the unmarshal path.
func TestReactionTypeEmoji_RoundTrip(t *testing.T) {
	in := &ReactionTypeEmoji{
		Type:  ReactionTypeKindEmoji,
		Emoji: ReactionEmojiThumbsUp,
	}
	data, err := json.Marshal(in)
	require.NoError(t, err)
	require.Contains(t, string(data), `"emoji":"👍"`)

	var out ReactionTypeEmoji
	require.NoError(t, json.Unmarshal(data, &out))
	require.Equal(t, ReactionEmojiThumbsUp, out.Emoji)
}
