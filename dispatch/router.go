package dispatch

import (
	"context"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/lukaszraczylo/go-telegram/api"
	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/lukaszraczylo/go-telegram/transport"
)

// Router dispatches updates from any Updater to typed handlers.
//
// Matchers run in registration order; first match wins. A panic-recovery
// middleware is attached automatically and runs around every dispatch.
type Router struct {
	bot *client.Bot

	commands           []commandRoute
	texts              []textRoute
	callbacks          []callbackRoute
	inlines            []Handler[*api.InlineQuery]
	editedMsg          []Handler[*api.Message]
	channelPosts       []Handler[*api.Message]
	editedChannelPosts []Handler[*api.Message]

	messageFilters  []messageFilterRoute
	callbackFilters []callbackFilterRoute
	inlineFilters   []inlineFilterRoute

	// typed update handlers
	myChatMember       []Handler[*api.ChatMemberUpdated]
	chatMember         []Handler[*api.ChatMemberUpdated]
	chatJoinRequest    []Handler[*api.ChatJoinRequest]
	preCheckoutQuery   []Handler[*api.PreCheckoutQuery]
	shippingQuery      []Handler[*api.ShippingQuery]
	poll               []Handler[*api.Poll]
	pollAnswer         []Handler[*api.PollAnswer]
	chosenInlineResult []Handler[*api.ChosenInlineResult]
	messageReaction    []Handler[*api.MessageReactionUpdated]
	messageReactionCnt []Handler[*api.MessageReactionCountUpdated]
	chatBoost          []Handler[*api.ChatBoostUpdated]
	removedChatBoost   []Handler[*api.ChatBoostRemoved]
	businessConn       []Handler[*api.BusinessConnection]
	purchasedPaidMedia []Handler[*api.PaidMediaPurchased]

	myChatMemberFilters    []chatMemberFilterRoute
	chatMemberFilters      []chatMemberFilterRoute
	chatJoinRequestFilters []chatJoinRequestFilterRoute
	preCheckoutFilters     []preCheckoutFilterRoute

	// group-priority routes (registered via Router.Group())
	groupCommands       []groupCommandRoute
	groupTexts          []groupTextRoute
	groupMessageFilters []groupMessageFilterRoute

	globalMW []Middleware[*api.Update]

	maxConcurrency int // default 50; 0 = serial (legacy)
	sem            chan struct{}
}

type messageFilterRoute struct {
	filter  Filter[*api.Message]
	handler Handler[*api.Message]
}

type callbackFilterRoute struct {
	filter  Filter[*api.CallbackQuery]
	handler Handler[*api.CallbackQuery]
}

type inlineFilterRoute struct {
	filter  Filter[*api.InlineQuery]
	handler Handler[*api.InlineQuery]
}

type chatMemberFilterRoute struct {
	filter  Filter[*api.ChatMemberUpdated]
	handler Handler[*api.ChatMemberUpdated]
}

type chatJoinRequestFilterRoute struct {
	filter  Filter[*api.ChatJoinRequest]
	handler Handler[*api.ChatJoinRequest]
}

type preCheckoutFilterRoute struct {
	filter  Filter[*api.PreCheckoutQuery]
	handler Handler[*api.PreCheckoutQuery]
}

// RouterOption configures a Router at construction time.
type RouterOption func(*Router)

// WithMaxConcurrency sets the maximum number of updates processed in parallel.
// Default is 50. Pass 0 to dispatch serially (one update at a time, in the
// calling goroutine — the legacy behaviour before v1.1.0).
//
// Note: concurrent dispatch means handlers for different updates may run
// simultaneously. Handlers that mutate shared state must be safe for concurrent
// access.
func WithMaxConcurrency(n int) RouterOption {
	return func(r *Router) { r.maxConcurrency = n }
}

type commandRoute struct {
	cmd     string
	handler Handler[*api.Message]
}

type textRoute struct {
	re      *regexp.Regexp
	handler Handler[*api.Message]
}

type callbackRoute struct {
	re      *regexp.Regexp
	handler Handler[*api.CallbackQuery]
}

// New constructs a Router. Recovery middleware is added by default; users
// can disable it by passing WithoutRecovery (not implemented here, but
// the hook is in place via Use).
func New(b *client.Bot, opts ...RouterOption) *Router {
	r := &Router{bot: b, maxConcurrency: 50}
	for _, o := range opts {
		o(r)
	}
	if r.maxConcurrency > 0 {
		r.sem = make(chan struct{}, r.maxConcurrency)
	}
	r.Use(Recovery())
	return r
}

// Use registers a global middleware applied to every Update dispatch.
func (r *Router) Use(mw Middleware[*api.Update]) { r.globalMW = append(r.globalMW, mw) }

// OnCommand registers a handler for a slash command. The command string
// includes the leading slash (e.g. "/start"). Matching strips an optional
// "@BotName" suffix.
func (r *Router) OnCommand(cmd string, h Handler[*api.Message]) {
	r.commands = append(r.commands, commandRoute{cmd: cmd, handler: h})
}

// OnText registers a handler for messages whose Text matches the regex.
//
// Panics at registration time if pattern is not a valid regular expression.
func (r *Router) OnText(pattern string, h Handler[*api.Message]) {
	r.texts = append(r.texts, textRoute{re: regexp.MustCompile(pattern), handler: h})
}

// OnCallback registers a handler for callback queries whose Data matches
// the regex.
//
// Panics at registration time if pattern is not a valid regular expression.
func (r *Router) OnCallback(pattern string, h Handler[*api.CallbackQuery]) {
	r.callbacks = append(r.callbacks, callbackRoute{re: regexp.MustCompile(pattern), handler: h})
}

// OnInlineQuery registers a handler for inline queries (one matcher only;
// inline queries are not partitioned by content here).
func (r *Router) OnInlineQuery(h Handler[*api.InlineQuery]) {
	r.inlines = append(r.inlines, h)
}

// OnEditedMessage registers a handler for edited message updates.
func (r *Router) OnEditedMessage(h Handler[*api.Message]) {
	r.editedMsg = append(r.editedMsg, h)
}

// OnChannelPost registers a handler for channel post updates.
func (r *Router) OnChannelPost(h Handler[*api.Message]) {
	r.channelPosts = append(r.channelPosts, h)
}

// OnEditedChannelPost registers a handler for edited channel post updates.
func (r *Router) OnEditedChannelPost(h Handler[*api.Message]) {
	r.editedChannelPosts = append(r.editedChannelPosts, h)
}

// OnMessageFilter registers a typed message handler gated by filter f.
// Filter routes are checked after command and text routes; first match wins.
func (r *Router) OnMessageFilter(f Filter[*api.Message], h Handler[*api.Message]) {
	r.messageFilters = append(r.messageFilters, messageFilterRoute{filter: f, handler: h})
}

// OnCallbackFilter registers a typed callback-query handler gated by filter f.
// Filter routes are checked after pattern-based OnCallback routes; first match wins.
func (r *Router) OnCallbackFilter(f Filter[*api.CallbackQuery], h Handler[*api.CallbackQuery]) {
	r.callbackFilters = append(r.callbackFilters, callbackFilterRoute{filter: f, handler: h})
}

// OnInlineQueryFilter registers an inline-query handler gated by filter f.
// Filter routes are checked after bare OnInlineQuery handlers; first match wins.
func (r *Router) OnInlineQueryFilter(f Filter[*api.InlineQuery], h Handler[*api.InlineQuery]) {
	r.inlineFilters = append(r.inlineFilters, inlineFilterRoute{filter: f, handler: h})
}

// OnMyChatMember registers a handler for bot's own chat member status changes.
func (r *Router) OnMyChatMember(h Handler[*api.ChatMemberUpdated]) {
	r.myChatMember = append(r.myChatMember, h)
}

// OnMyChatMemberFilter registers a filtered handler for bot's own chat member status changes.
func (r *Router) OnMyChatMemberFilter(f Filter[*api.ChatMemberUpdated], h Handler[*api.ChatMemberUpdated]) {
	r.myChatMemberFilters = append(r.myChatMemberFilters, chatMemberFilterRoute{filter: f, handler: h})
}

// OnChatMember registers a handler for chat member status changes.
func (r *Router) OnChatMember(h Handler[*api.ChatMemberUpdated]) {
	r.chatMember = append(r.chatMember, h)
}

// OnChatMemberFilter registers a filtered handler for chat member status changes.
func (r *Router) OnChatMemberFilter(f Filter[*api.ChatMemberUpdated], h Handler[*api.ChatMemberUpdated]) {
	r.chatMemberFilters = append(r.chatMemberFilters, chatMemberFilterRoute{filter: f, handler: h})
}

// OnChatJoinRequest registers a handler for chat join requests.
func (r *Router) OnChatJoinRequest(h Handler[*api.ChatJoinRequest]) {
	r.chatJoinRequest = append(r.chatJoinRequest, h)
}

// OnChatJoinRequestFilter registers a filtered handler for chat join requests.
func (r *Router) OnChatJoinRequestFilter(f Filter[*api.ChatJoinRequest], h Handler[*api.ChatJoinRequest]) {
	r.chatJoinRequestFilters = append(r.chatJoinRequestFilters, chatJoinRequestFilterRoute{filter: f, handler: h})
}

// OnPreCheckoutQuery registers a handler for pre-checkout queries.
func (r *Router) OnPreCheckoutQuery(h Handler[*api.PreCheckoutQuery]) {
	r.preCheckoutQuery = append(r.preCheckoutQuery, h)
}

// OnPreCheckoutQueryFilter registers a filtered handler for pre-checkout queries.
func (r *Router) OnPreCheckoutQueryFilter(f Filter[*api.PreCheckoutQuery], h Handler[*api.PreCheckoutQuery]) {
	r.preCheckoutFilters = append(r.preCheckoutFilters, preCheckoutFilterRoute{filter: f, handler: h})
}

// OnShippingQuery registers a handler for shipping queries.
func (r *Router) OnShippingQuery(h Handler[*api.ShippingQuery]) {
	r.shippingQuery = append(r.shippingQuery, h)
}

// OnPoll registers a handler for poll state updates.
func (r *Router) OnPoll(h Handler[*api.Poll]) {
	r.poll = append(r.poll, h)
}

// OnPollAnswer registers a handler for poll answer updates.
func (r *Router) OnPollAnswer(h Handler[*api.PollAnswer]) {
	r.pollAnswer = append(r.pollAnswer, h)
}

// OnChosenInlineResult registers a handler for chosen inline results.
func (r *Router) OnChosenInlineResult(h Handler[*api.ChosenInlineResult]) {
	r.chosenInlineResult = append(r.chosenInlineResult, h)
}

// OnMessageReaction registers a handler for message reaction updates.
func (r *Router) OnMessageReaction(h Handler[*api.MessageReactionUpdated]) {
	r.messageReaction = append(r.messageReaction, h)
}

// OnMessageReactionCount registers a handler for anonymous message reaction count updates.
func (r *Router) OnMessageReactionCount(h Handler[*api.MessageReactionCountUpdated]) {
	r.messageReactionCnt = append(r.messageReactionCnt, h)
}

// OnChatBoost registers a handler for chat boost updates.
func (r *Router) OnChatBoost(h Handler[*api.ChatBoostUpdated]) {
	r.chatBoost = append(r.chatBoost, h)
}

// OnRemovedChatBoost registers a handler for removed chat boost updates.
func (r *Router) OnRemovedChatBoost(h Handler[*api.ChatBoostRemoved]) {
	r.removedChatBoost = append(r.removedChatBoost, h)
}

// OnBusinessConnection registers a handler for business connection updates.
func (r *Router) OnBusinessConnection(h Handler[*api.BusinessConnection]) {
	r.businessConn = append(r.businessConn, h)
}

// OnPurchasedPaidMedia registers a handler for purchased paid media updates.
func (r *Router) OnPurchasedPaidMedia(h Handler[*api.PaidMediaPurchased]) {
	r.purchasedPaidMedia = append(r.purchasedPaidMedia, h)
}

// Run consumes the Updater and dispatches each update. It blocks until
// the Updater's channel is closed or ctx is cancelled.
//
// By default updates are processed concurrently (up to WithMaxConcurrency(50)
// goroutines). Handlers for different updates may therefore run simultaneously;
// shared state must be protected. Pass WithMaxConcurrency(0) to New to restore
// serial (legacy) behaviour.
//
// Run waits for all in-flight handlers to finish before returning.
// Process runs a single update through the router's middleware and handler
// chain synchronously. Entry point for callers sourcing updates outside the
// standard transport.Updater flow — custom webhook frameworks, message-bus
// consumers, or tests driving the router without spinning up Run.
//
// Honours the router's global middleware (Use) but bypasses the concurrency
// semaphore wired up by Run; the caller controls parallelism.
func (r *Router) Process(ctx context.Context, u *api.Update) error {
	if u == nil {
		return nil
	}
	root := r.dispatch
	for i := len(r.globalMW) - 1; i >= 0; i-- {
		root = r.globalMW[i](root)
	}
	c := NewContext(ctx, r.bot, u)
	return root(c, u)
}

func (r *Router) Run(ctx context.Context, u transport.Updater) error {
	runErr := make(chan error, 1)
	go func() { runErr <- u.Run(ctx) }()

	root := r.dispatch
	for i := len(r.globalMW) - 1; i >= 0; i-- {
		root = r.globalMW[i](root)
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	dispatch := func(up api.Update) {
		c := NewContext(ctx, r.bot, &up)
		if err := root(c, &up); err != nil {
			if r.bot != nil {
				r.bot.Logger().Error("dispatch handler error", "err", err, "update_id", up.UpdateID)
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-runErr:
			return err
		case up, ok := <-u.Updates():
			if !ok {
				// Channel closed; consume the run error if pending.
				select {
				case err := <-runErr:
					return err
				default:
				}
				return nil
			}

			if r.sem == nil {
				// Serial mode (legacy / WithMaxConcurrency(0)).
				dispatch(up)
				continue
			}

			// Concurrent mode: acquire semaphore slot then launch goroutine.
			select {
			case r.sem <- struct{}{}:
			case <-ctx.Done():
				return ctx.Err()
			}
			wg.Add(1)
			go func(up api.Update) {
				defer func() {
					<-r.sem
					wg.Done()
				}()
				dispatch(up)
			}(up)
		}
	}
}

func (r *Router) dispatch(c *Context, u *api.Update) error {
	switch {
	case u.Message != nil:
		return r.handleMessage(c, u.Message)
	case u.EditedMessage != nil:
		return runHandlers(r.editedMsg, c, u.EditedMessage)
	case u.ChannelPost != nil:
		return runHandlers(r.channelPosts, c, u.ChannelPost)
	case u.EditedChannelPost != nil:
		return runHandlers(r.editedChannelPosts, c, u.EditedChannelPost)
	case u.CallbackQuery != nil:
		return r.handleCallback(c, u.CallbackQuery)
	case u.InlineQuery != nil:
		if err := runHandlers(r.inlines, c, u.InlineQuery); err != nil {
			return err
		}
		for _, route := range r.inlineFilters {
			if route.filter(u.InlineQuery) {
				return route.handler(c, u.InlineQuery)
			}
		}
		return nil
	case u.MyChatMember != nil:
		return r.handleChatMemberUpdate(c, u.MyChatMember, r.myChatMember, r.myChatMemberFilters)
	case u.ChatMember != nil:
		return r.handleChatMemberUpdate(c, u.ChatMember, r.chatMember, r.chatMemberFilters)
	case u.ChatJoinRequest != nil:
		return r.handleChatJoinRequest(c, u.ChatJoinRequest)
	case u.PreCheckoutQuery != nil:
		return r.handlePreCheckoutQuery(c, u.PreCheckoutQuery)
	case u.ShippingQuery != nil:
		return runHandlers(r.shippingQuery, c, u.ShippingQuery)
	case u.Poll != nil:
		return runHandlers(r.poll, c, u.Poll)
	case u.PollAnswer != nil:
		return runHandlers(r.pollAnswer, c, u.PollAnswer)
	case u.ChosenInlineResult != nil:
		return runHandlers(r.chosenInlineResult, c, u.ChosenInlineResult)
	case u.MessageReaction != nil:
		return runHandlers(r.messageReaction, c, u.MessageReaction)
	case u.MessageReactionCount != nil:
		return runHandlers(r.messageReactionCnt, c, u.MessageReactionCount)
	case u.ChatBoost != nil:
		return runHandlers(r.chatBoost, c, u.ChatBoost)
	case u.RemovedChatBoost != nil:
		return runHandlers(r.removedChatBoost, c, u.RemovedChatBoost)
	case u.BusinessConnection != nil:
		return runHandlers(r.businessConn, c, u.BusinessConnection)
	case u.PurchasedPaidMedia != nil:
		return runHandlers(r.purchasedPaidMedia, c, u.PurchasedPaidMedia)
	}
	return nil
}

func (r *Router) handleChatMemberUpdate(c *Context, payload *api.ChatMemberUpdated, handlers []Handler[*api.ChatMemberUpdated], filters []chatMemberFilterRoute) error {
	if err := runHandlers(handlers, c, payload); err != nil {
		return err
	}
	for _, route := range filters {
		if route.filter(payload) {
			return route.handler(c, payload)
		}
	}
	return nil
}

func (r *Router) handleChatJoinRequest(c *Context, payload *api.ChatJoinRequest) error {
	if err := runHandlers(r.chatJoinRequest, c, payload); err != nil {
		return err
	}
	for _, route := range r.chatJoinRequestFilters {
		if route.filter(payload) {
			return route.handler(c, payload)
		}
	}
	return nil
}

func (r *Router) handlePreCheckoutQuery(c *Context, payload *api.PreCheckoutQuery) error {
	if err := runHandlers(r.preCheckoutQuery, c, payload); err != nil {
		return err
	}
	for _, route := range r.preCheckoutFilters {
		if route.filter(payload) {
			return route.handler(c, payload)
		}
	}
	return nil
}

// runHandlers invokes each handler in order; returns the first non-nil error.
func runHandlers[T any](handlers []Handler[T], c *Context, payload T) error {
	for _, h := range handlers {
		if err := h(c, payload); err != nil {
			return err
		}
	}
	return nil
}

func (r *Router) handleMessage(c *Context, m *api.Message) error {
	// Try command first (entity-aware).
	if cmd, args, ok := extractCommand(m); ok {
		for _, route := range r.commands {
			if route.cmd == cmd {
				c.Command = cmd
				c.CommandArgs = args
				return route.handler(c, m)
			}
		}
	}
	// Then text regex matchers.
	if m.Text != "" {
		for _, route := range r.texts {
			if subs := route.re.FindStringSubmatch(m.Text); subs != nil {
				c.RegexMatch = subs
				return route.handler(c, m)
			}
		}
	}
	// Filter-based routes.
	for _, route := range r.messageFilters {
		if route.filter(m) {
			return route.handler(c, m)
		}
	}
	// Group-priority routes (registered via RouterScope.Group()).
	return r.dispatchGroups(c, m)
}

func (r *Router) handleCallback(c *Context, q *api.CallbackQuery) error {
	for _, route := range r.callbacks {
		if subs := route.re.FindStringSubmatch(q.Data); subs != nil {
			c.RegexMatch = subs
			return route.handler(c, q)
		}
	}
	// Filter-based routes checked after pattern routes.
	for _, route := range r.callbackFilters {
		if route.filter(q) {
			return route.handler(c, q)
		}
	}
	return nil
}

// extractCommand returns the command (e.g. "/start") and the remaining
// argument string, when m carries a leading bot_command entity. It strips
// optional "@BotName" suffix on the command itself.
func extractCommand(m *api.Message) (cmd, args string, ok bool) {
	if len(m.Entities) == 0 || m.Text == "" {
		return "", "", false
	}
	first := m.Entities[0]
	if first.Type != api.MessageEntityTypeBotCommand || first.Offset != 0 {
		return "", "", false
	}
	cmd, sliceOk := utf16Slice(m.Text, int(first.Offset), int(first.Length))
	if !sliceOk {
		return "", "", false
	}
	if i := strings.Index(cmd, "@"); i >= 0 {
		cmd = cmd[:i]
	}
	end := int(first.Offset) + int(first.Length)
	rest, _ := utf16Slice(m.Text, end, utf16Len(m.Text)-end)
	args = strings.TrimSpace(rest)
	return cmd, args, true
}

// utf16Slice returns the substring of s identified by a UTF-16 offset/length
// pair, as Telegram's MessageEntity uses. ok is false if the indices fall
// outside s's UTF-16 length.
func utf16Slice(s string, offset, length int) (string, bool) {
	runes := []rune(s)
	var startBytes, endBytes int
	var u16 int
	found := false
	for i, r := range runes {
		if u16 == offset {
			startBytes = byteIndex(runes, i)
			found = true
		}
		if u16 == offset+length {
			endBytes = byteIndex(runes, i)
			return s[startBytes:endBytes], true
		}
		if r > 0xFFFF {
			u16 += 2
		} else {
			u16++
		}
	}
	if found && u16 == offset+length {
		return s[startBytes:], true
	}
	return "", false
}

func byteIndex(runes []rune, runeIdx int) int {
	n := 0
	for i := 0; i < runeIdx; i++ {
		n += utf8.RuneLen(runes[i])
	}
	return n
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}
