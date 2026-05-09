// Package main demonstrates creating polls and tallying answers via OnPollAnswer.
//
// Usage: send "/poll <question>" in a group or private chat.
// The bot creates a non-anonymous poll with four preset options and tallies votes.
//
//	TELEGRAM_BOT_TOKEN=xxx go run ./examples/polls
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/dispatch"
	"github.com/lukaszraczylo/go-telegram/transport"
)

// pollTally maps pollID → optionIndex → voteCount.
type pollTally struct {
	mu    sync.Mutex
	votes map[string]map[int64]int
}

func newPollTally() *pollTally {
	return &pollTally{votes: make(map[string]map[int64]int)}
}

func (t *pollTally) record(pollID string, opts []int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.votes[pollID] == nil {
		t.votes[pollID] = make(map[int64]int)
	}
	for _, opt := range opts {
		t.votes[pollID][opt]++
	}
}

func (t *pollTally) summary(pollID string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	m := t.votes[pollID]
	if len(m) == 0 {
		return "No votes yet."
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Tally for poll %s:\n", pollID)
	for opt, count := range m {
		fmt.Fprintf(&sb, "  Option %d: %d vote(s)\n", opt, count)
	}
	return sb.String()
}

var pollOptions = []api.InputPollOption{
	{Text: "Option A"},
	{Text: "Option B"},
	{Text: "Option C"},
	{Text: "Option D"},
}

func main() {
	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bot := client.New(token,
		client.WithHTTPClient(client.NewRetryDoer(client.NewDefaultHTTPDoer())),
	)

	tally := newPollTally()
	router := dispatch.New(bot)

	router.OnCommand("/poll", func(c *dispatch.Context, m *api.Message) error {
		question := strings.TrimSpace(m.Text)
		// Strip the "/poll" command prefix.
		if after, ok := strings.CutPrefix(question, "/poll"); ok {
			question = strings.TrimSpace(after)
		}
		if question == "" {
			_, _ = api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
				ChatID: api.ChatIDFromInt(m.Chat.ID),
				Text:   "Usage: /poll <question>",
			})
			return nil
		}
		msg, err := api.SendPoll(c.Ctx, c.Bot, &api.SendPollParams{
			ChatID:      api.ChatIDFromInt(m.Chat.ID),
			Question:    question,
			Options:     pollOptions,
			IsAnonymous: api.Ptr(false),
		})
		if err != nil {
			return err
		}
		if msg != nil && msg.Poll != nil {
			log.Printf("poll created: id=%s question=%q", msg.Poll.ID, msg.Poll.Question)
		}
		return nil
	})

	router.OnCommand("/tally", func(c *dispatch.Context, m *api.Message) error {
		pollID := strings.TrimSpace(m.Text)
		if after, ok := strings.CutPrefix(pollID, "/tally"); ok {
			pollID = strings.TrimSpace(after)
		}
		if pollID == "" {
			_, _ = api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
				ChatID: api.ChatIDFromInt(m.Chat.ID),
				Text:   "Usage: /tally <poll_id>",
			})
			return nil
		}
		_, _ = api.SendMessage(c.Ctx, c.Bot, &api.SendMessageParams{
			ChatID: api.ChatIDFromInt(m.Chat.ID),
			Text:   tally.summary(pollID),
		})
		return nil
	})

	router.OnPollAnswer(func(c *dispatch.Context, pa *api.PollAnswer) error {
		userID := int64(0)
		if pa.User != nil {
			userID = pa.User.ID
		}
		log.Printf("poll answer: poll=%s user=%d options=%v", pa.PollID, userID, pa.OptionIds)
		tally.record(pa.PollID, pa.OptionIds)
		return nil
	})

	poller := transport.NewLongPoller(bot)
	if err := router.Run(ctx, poller); err != nil && err != context.Canceled {
		log.Printf("router exited: %v", err)
	}
}
