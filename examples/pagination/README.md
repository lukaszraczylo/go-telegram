# pagination

Multi-page inline keyboard for browsing a list. No server-side session required — page state is encoded in callback data.

## What it shows

- `OnCommand("/list")` sends the first page with inline navigation buttons
- `OnCallback("^page:(\\d+)$")` parses page number from callback data via `c.Values["regex_match"]`
- `api.EditMessageText` edits the message in-place on each page turn
- `api.AnswerCallbackQuery` dismisses the loading spinner

## Pattern

Callback data format: `page:<n>` where `n` is the 0-based page index.

The `renderPage` helper builds both the text content and the keyboard in one call. Only [« Prev] or [Next »] buttons that make sense for the current page are rendered, so the keyboard is always minimal.

## Running

```bash
export TELEGRAM_BOT_TOKEN=123456:ABC...
go run ./examples/pagination
```

Send `/list` to the bot. Tap Next/Prev to navigate 20 sample items, 5 per page.

## Extending

To paginate dynamic data (database results, API responses), replace `sampleItems` with a function that takes `(page, pageSize)` and returns items + total count.
