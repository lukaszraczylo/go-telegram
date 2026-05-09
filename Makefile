.PHONY: test test-race lint vet integration regen snapshot regen-from-fixture test-update-golden clean clean-generated audit audit-drift docs docs-check help

GO ?= go

help:
	@echo "Targets:"
	@echo "  test                 - run unit tests"
	@echo "  test-race            - run unit tests with race detector"
	@echo "  lint                 - go vet + staticcheck"
	@echo "  integration          - run integration suite (requires TELEGRAM_BOT_TOKEN)"
	@echo "  snapshot             - capture HTML snapshot from live API (Plan 2)"
	@echo "  regen                - regenerate api/ from latest snapshot (Plan 2)"
	@echo "  regen-from-fixture   - deterministic regen from pinned fixture (Plan 2)"
	@echo "  test-update-golden   - refresh golden test fixtures (Plan 2)"
	@echo "  audit                - report any-typed/bool fallbacks in current IR"
	@echo "  audit-drift          - audit + compare against HEAD's IR for signature changes"
	@echo "  docs                 - regenerate markdown reference docs into docs/reference/"
	@echo "  docs-check           - assert docs/reference/ is up to date (CI gate)"
	@echo "  clean-generated      - delete generated api/*.gen.go and internal/spec/api.json"
	@echo "  clean                - clean-generated + transient artefacts (binaries, coverage)"

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

lint: vet
	@which staticcheck > /dev/null || (echo "install staticcheck: go install honnef.co/go/tools/cmd/staticcheck@latest" && exit 1)
	staticcheck ./...

integration:
	$(GO) test -tags=integration -v ./test/integration/...

SCRAPE_INPUT ?= testdata/html/snapshot_2026-05-08.html
SCRAPE_OUTPUT ?= internal/spec/api.json

snapshot:
	./scripts/snapshot.sh

regen: clean-generated
	$(GO) run ./cmd/scrape -input testdata/html/latest.html -output $(SCRAPE_OUTPUT)
	$(GO) run ./cmd/audit -ir $(SCRAPE_OUTPUT)
	$(GO) run ./cmd/genapi -input $(SCRAPE_OUTPUT) -outdir api
	$(GO) test ./api/...
	$(MAKE) docs

regen-from-fixture: clean-generated
	$(GO) run ./cmd/scrape -input $(SCRAPE_INPUT) -output $(SCRAPE_OUTPUT)
	$(GO) run ./cmd/audit -ir $(SCRAPE_OUTPUT)
	$(GO) run ./cmd/genapi -input $(SCRAPE_OUTPUT) -outdir api
	$(GO) test ./api/...
	$(MAKE) docs

audit:
	$(GO) run ./cmd/audit -ir $(SCRAPE_OUTPUT)

audit-drift:
	$(GO) run ./cmd/audit -ir $(SCRAPE_OUTPUT) -drift

test-update-golden:
	$(GO) test -run TestEmit -update ./cmd/genapi/...
	$(GO) test -run TestScrape -update ./cmd/scrape/...

# Regenerate godoc-style markdown reference docs into docs/reference/.
# Auto-installs gomarkdoc on first run.
DOC_PACKAGES := \
	./client \
	./transport \
	./dispatch \
	./dispatch/conversation \
	./dispatch/filters/message \
	./dispatch/filters/callback \
	./dispatch/filters/inline \
	./dispatch/filters/chatmember \
	./dispatch/filters/chatjoinrequest \
	./dispatch/filters/precheckoutquery \
	./api

docs:
	@which gomarkdoc > /dev/null || (echo "installing gomarkdoc..." && $(GO) install github.com/princjef/gomarkdoc/cmd/gomarkdoc@v1.1.0)
	gomarkdoc \
		--repository.url=https://github.com/lukaszraczylo/go-telegram \
		--repository.default-branch=main \
		--repository.path=/ \
		-o 'docs/reference/{{.Dir}}.md' $(DOC_PACKAGES)

docs-check: docs
	@git diff --exit-code docs/reference/ || (echo "docs/reference/ is stale — run 'make docs' and commit" && exit 1)

# clean-generated removes ONLY codegen output. Source code (cmd/scrape,
# cmd/genapi, runtime helpers) is untouched. Run before regen to avoid
# orphan files lingering when the IR shrinks (renamed/removed methods).
clean-generated:
	rm -f api/*.gen.go api/*_gen_test.go
	rm -f internal/spec/api.json

# clean removes generated output AND transient artefacts (binaries
# accidentally left at repo root, coverage reports). Source code is
# never touched.
clean: clean-generated
	rm -f coverage.out coverage.html
	rm -f echo webhook genapi scrape callback files inline conversation middleware stateful
