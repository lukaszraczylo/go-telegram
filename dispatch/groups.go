package dispatch

import (
	"errors"
	"regexp"
	"sort"

	"github.com/lukaszraczylo/go-telegram/api"
)

// ErrEndGroups stops dispatch from running any further handlers in any
// group for this update when returned by a handler. Use it to indicate
// the update has been definitively handled.
//
// errors.Is(err, ErrEndGroups) is the canonical check, though dispatch
// itself recognises it by exact identity.
var ErrEndGroups = errors.New("dispatch: end groups")

// ErrContinueGroups signals that this group's handler should be treated
// as not-matching when returned by a handler: dispatch moves on to the
// next handler in the same group, then to subsequent groups.
//
// Without ErrContinueGroups, a non-error return from a matched handler
// stops dispatch (default first-match-wins semantics).
var ErrContinueGroups = errors.New("dispatch: continue groups")

// RouterScope registers handlers into a specific priority group on its parent
// Router. Group 0 runs first, then group 1, etc. Within a group, handlers run
// in registration order; the first non-skipped match terminates dispatch
// unless the handler returns ErrContinueGroups.
type RouterScope struct {
	router *Router
	group  int
}

// Group returns a RouterScope that registers handlers in the given group.
// Group 0 (the default) runs first, then group 1, etc. Within a group,
// handlers run in registration order; the first non-skipped match
// terminates dispatch unless the handler returns ErrContinueGroups.
func (r *Router) Group(group int) *RouterScope {
	return &RouterScope{router: r, group: group}
}

// OnCommand registers a command handler in this group.
func (s *RouterScope) OnCommand(cmd string, h Handler[*api.Message]) {
	s.router.groupCommands = append(s.router.groupCommands, groupCommandRoute{
		cmd: cmd, group: s.group, handler: h,
	})
}

// OnText registers a regex text handler in this group.
// Panics at registration time if pattern is not a valid regular expression.
func (s *RouterScope) OnText(pattern string, h Handler[*api.Message]) {
	s.router.groupTexts = append(s.router.groupTexts, groupTextRoute{
		re: regexp.MustCompile(pattern), group: s.group, handler: h,
	})
}

// OnMessageFilter registers a filter-based message handler in this group.
func (s *RouterScope) OnMessageFilter(f Filter[*api.Message], h Handler[*api.Message]) {
	s.router.groupMessageFilters = append(s.router.groupMessageFilters, groupMessageFilterRoute{
		filter: f, group: s.group, handler: h,
	})
}

// group-aware route types

type groupCommandRoute struct {
	cmd     string
	group   int
	handler Handler[*api.Message]
}

type groupTextRoute struct {
	re      *regexp.Regexp
	group   int
	handler Handler[*api.Message]
}

type groupMessageFilterRoute struct {
	filter  Filter[*api.Message]
	group   int
	handler Handler[*api.Message]
}

// dispatchGroups runs message handlers registered via RouterScope.Group().
// It collects all matching groups, sorts by group number, and applies
// first-match-wins semantics within each group. Handlers may return
// ErrContinueGroups (skip to next handler/group) or ErrEndGroups (stop all groups).
// A non-sentinel error stops dispatch and is returned to the caller.
func (r *Router) dispatchGroups(c *Context, m *api.Message) error {
	// Collect group numbers present.
	groupSet := map[int]struct{}{}
	for _, gr := range r.groupCommands {
		groupSet[gr.group] = struct{}{}
	}
	for _, gr := range r.groupTexts {
		groupSet[gr.group] = struct{}{}
	}
	for _, gr := range r.groupMessageFilters {
		groupSet[gr.group] = struct{}{}
	}
	if len(groupSet) == 0 {
		return nil
	}

	groups := make([]int, 0, len(groupSet))
	for g := range groupSet {
		groups = append(groups, g)
	}
	sort.Ints(groups)

	for _, g := range groups {
		matched, err := r.runGroupHandlers(c, m, g)
		if err != nil {
			if errors.Is(err, ErrEndGroups) {
				return nil
			}
			return err
		}
		if matched {
			// First-match-wins: stop further groups.
			return nil
		}
		// No match or ErrContinueGroups from all handlers: try next group.
	}
	return nil
}

// runGroupHandlers runs all handlers in group g against m, in registration
// order. Returns (true, nil) when a handler matched (returned nil). Returns
// (false, nil) when all handlers returned ErrContinueGroups. Returns
// (false, err) for ErrEndGroups or any non-sentinel error.
func (r *Router) runGroupHandlers(c *Context, m *api.Message, g int) (matched bool, err error) {
	// Commands.
	if cmd, args, ok := extractCommand(m); ok {
		for _, route := range r.groupCommands {
			if route.group != g || route.cmd != cmd {
				continue
			}
			c.Values["command"] = cmd
			c.Values["command_args"] = args
			if err := route.handler(c, m); err != nil {
				if errors.Is(err, ErrContinueGroups) {
					continue
				}
				return false, err
			}
			return true, nil
		}
	}
	// Text regex.
	if m.Text != "" {
		for _, route := range r.groupTexts {
			if route.group != g {
				continue
			}
			subs := route.re.FindStringSubmatch(m.Text)
			if subs == nil {
				continue
			}
			c.Values["regex_match"] = subs
			if err := route.handler(c, m); err != nil {
				if errors.Is(err, ErrContinueGroups) {
					continue
				}
				return false, err
			}
			return true, nil
		}
	}
	// Filter-based.
	for _, route := range r.groupMessageFilters {
		if route.group != g || !route.filter(m) {
			continue
		}
		if err := route.handler(c, m); err != nil {
			if errors.Is(err, ErrContinueGroups) {
				continue
			}
			return false, err
		}
		return true, nil
	}
	return false, nil
}
