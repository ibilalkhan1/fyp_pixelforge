.PHONY: test
test:
	go test -race -v ./...

SRC := $(shell find -name main.go)

.PHONY: build
build: $(SRC)
	for main in $(SRC) ; do \
		go build $$main ; \
	done
	rm main

.PHONY: lint
lint:
	golangci-lint run

# playerbins cross-compiles the universal pixelforge-player for
# every shipping target and stages the result under
# pixelforge_studio/playerbins/bins/<goos>-<goarch>/ so go:embed
# picks them up. The release pipeline runs this before tagging so
# the shipped studio carries up-to-date players + day-one users
# (no Go toolchain) can Build → Host out of the box.
#
# CGO_ENABLED=1 is implicit for the native targets — Ebitengine's
# desktop backend needs cgo to link against the host's audio +
# graphics libraries. WASM is pure-Go and runs with CGO_ENABLED=0.
.PHONY: playerbins
playerbins:
	@mkdir -p pixelforge_studio/playerbins/bins/linux-amd64
	@mkdir -p pixelforge_studio/playerbins/bins/darwin-amd64
	@mkdir -p pixelforge_studio/playerbins/bins/darwin-arm64
	@mkdir -p pixelforge_studio/playerbins/bins/windows-amd64
	@mkdir -p pixelforge_studio/playerbins/bins/js-wasm
	GOOS=linux GOARCH=amd64 go build -tags=long -o pixelforge_studio/playerbins/bins/linux-amd64/pixelforge-player ./cmd/pixelforge-player
	GOOS=darwin GOARCH=amd64 go build -tags=long -o pixelforge_studio/playerbins/bins/darwin-amd64/pixelforge-player ./cmd/pixelforge-player
	GOOS=darwin GOARCH=arm64 go build -tags=long -o pixelforge_studio/playerbins/bins/darwin-arm64/pixelforge-player ./cmd/pixelforge-player
	GOOS=windows GOARCH=amd64 go build -tags=long -o pixelforge_studio/playerbins/bins/windows-amd64/pixelforge-player.exe ./cmd/pixelforge-player
	GOOS=js GOARCH=wasm go build -o pixelforge_studio/playerbins/bins/js-wasm/pixelforge-player.wasm ./cmd/pixelforge-player

# verb-catalog regenerates docs/verb-catalog.md from the verb-recipe
# registry (pixelforge_studio/scripting/catalog). Equivalent to
# `go generate ./pixelforge_studio/scripting/catalog/...` — exposed
# as a named target so CI + contributors can find it via
# `make help`-style discovery. Output is deterministic; the CI gate
# in plan-009 U25 verifies the file is current by re-running this
# and asserting no diff.
.PHONY: verb-catalog
verb-catalog:
	go run ./pixelforge_studio/scripting/catalog/cmd/gendocs -out docs/verb-catalog.md

# verb-coverage prints the catalog's used/unused topics across the
# four proof-game baseline fixtures. Informational only (per R19);
# does not fail when verbs are unused.
.PHONY: verb-coverage
verb-coverage:
	go run ./pixelforge_studio/scripting/catalog/cmd/coverage

# capability-matrix regenerates docs/reference-games-capability-matrix.md
# from a local `go test -tags=long -json` capture. Plan-009 U25's CI
# workflow performs the same regeneration on each push + uploads the
# result as a workflow artifact (per scope-guardian finding F-007 the
# regenerated matrix is NOT committed back from CI to avoid a
# commit-feedback-loop). This target lets contributors preview the
# matrix locally before pushing.
#
# Capture inputs first with:
#   go test -tags=long -json ./pixelforge_studio/integration_test/... \
#     > test-results-local.json
.PHONY: capability-matrix
capability-matrix:
	@if [ ! -f test-results-local.json ]; then \
	  echo "test-results-local.json missing; run:" 1>&2; \
	  echo "  go test -tags=long -json ./pixelforge_studio/integration_test/... > test-results-local.json" 1>&2; \
	  exit 1; \
	fi
	go run ./pixelforge_studio/scripting/catalog/cmd/matrix \
	  -in $$(go env GOOS)-$$(go env GOARCH)=test-results-local.json \
	  -out docs/reference-games-capability-matrix.md

# ci-local runs the full CI flow on the developer's machine for a
# sanity-check before pushing. Mirrors .github/workflows/long.yml step
# order so a green ci-local is a strong predictor of a green CI run.
# Skips the cross-platform matrix leg (only the host's leg runs) +
# skips wasm-browser tests (those live in build.yml's separate flow).
.PHONY: ci-local
ci-local:
	go vet ./...
	go test ./...
	go test -tags=long -json ./pixelforge_studio/integration_test/... > test-results-local.json || true
	$(MAKE) capability-matrix
	go run ./pixelforge_studio/scripting/catalog/cmd/gendocs -out docs/verb-catalog.md
	@echo "ci-local: complete. Inspect docs/reference-games-capability-matrix.md + docs/verb-catalog.md."

# playerbins-host is plan-009 U12's fast local-dev path: cross-
# compiling every supported OS is slow + needs cgo cross-toolchains
# the developer's machine usually lacks (linux amd64 cgo from a
# Darwin host, etc.). This target builds JUST the current host's
# native player + the WASM module — enough for Build → Host on the
# developer's own machine + the WASM smoke that runs on every host.
#
# Full cross-OS coverage stays in `make playerbins`; that's the
# release-pipeline target.
.PHONY: playerbins-host
playerbins-host:
	@HOST_GOOS=$$(go env GOOS); HOST_GOARCH=$$(go env GOARCH); \
	HOST_EXT=""; if [ "$$HOST_GOOS" = "windows" ]; then HOST_EXT=".exe"; fi; \
	mkdir -p pixelforge_studio/playerbins/bins/$$HOST_GOOS-$$HOST_GOARCH; \
	mkdir -p pixelforge_studio/playerbins/bins/js-wasm; \
	go build -tags=long -o pixelforge_studio/playerbins/bins/$$HOST_GOOS-$$HOST_GOARCH/pixelforge-player$$HOST_EXT ./cmd/pixelforge-player; \
	GOOS=js GOARCH=wasm go build -o pixelforge_studio/playerbins/bins/js-wasm/pixelforge-player.wasm ./cmd/pixelforge-player
