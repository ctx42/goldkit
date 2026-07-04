# AGENTS.md

Guidance for coding agents working in this repository. It follows the
[agents.md](https://agents.md) convention and is published with the repo.

## Overview

`goldkit` (module `github.com/ctx42/goldkit`) is a single-package Go library for
testing with **golden files** written in YAML. A golden file can hold plain
content, Go-templated content, or a full HTTP request/response exchange
(multipart included). All public API lives in `pkg/goldkit`.

## Commands

```bash
go test ./...                                            # Run all tests
go test -race ./...                                      # Run with the race detector (as CI does)
go test ./pkg/goldkit -run Test_NewRequest              # Run one top-level test
go test ./pkg/goldkit -run Test_NewRequest/without_data # Run one subtest
go vet ./...                                             # Vet
golangci-lint run -c tmp/.golangci.yml                  # Lint (config lives in tmp/, not repo root)
```

This is a library — there is no build target.

> **Note:** the lint config lives at `tmp/.golangci.yml`, which is `.gitignore`d,
> so the path must be passed explicitly and the file will not be present on a
> fresh clone.

## Continuous integration

CI runs on GitHub Actions (`.github/workflows/`):

- **`go.yml`** — on every push and pull request, runs `go test -race ./...` on
  `ubuntu-latest` with the Go version pinned by `go.mod`.
- **`docs.yaml`** — on pushes to `master` that touch `docs/**`, builds the Hugo
  documentation site and deploys it to GitHub Pages.

`go vet` and `golangci-lint` are run locally, not in CI.

## Architecture

The package models a golden file as a **body** plus a type-specific wrapper. Two
concepts are central.

**The `Body` interface** (`golden.go`) is the polymorphic core. Every body type
implements:

```go
Body() []byte                          // fresh slice each call
Assert(t tester.T, have []byte) bool   // true when have matches
SetContentTypeHeader(h http.Header)    // sets Content-Type
```

Implementations are selected by the `bodyType` YAML field via `parseBody`
(`helpers.go`); the type constants (`Text`, `JSON`, `Multipart`, `None`) are
exported from `golden.go`:

- `bodyText` (`body_text.go`) — default; exact string comparison, `text/plain`.
- `bodyJSON` (`body_json.go`) — semantic JSON equality (bytes need not match).
- `mpBody` (`body_mp.go`) — multipart; parses `files:`/`values:`, compares files
  byte-by-byte. Built on the standalone `MultiPart` builder (`multipart.go`).
- `bodyNone` (`body_none.go`) — asserts no body and sets no Content-Type.

Adding a new body type means: define the constant in `golden.go`, add a `case`
in `parseBody`, and implement the `Body` interface.

**Top-level golden file types** each embed `*base` (`base.go`, holding `Meta`,
`BodyType`, and the raw YAML `body` node) inline via `yaml:",inline"`, and each
follows the same lifecycle — `yaml.Unmarshal` into a struct with defaults, then
`setup(path)` (which calls `parseBody`, parses headers, and validates):

- `File` (`file.go`) — a bare golden file. `New(t, src)` / `Create(t, pth, data)`.
- `Request` (`request.go`) — one root `request:` field; builds `*http.Request`,
  defaulting scheme `http`, host `localhost`, bodyType `text`.
- `Response` (`response.go`) — one root `response:` field; builds `*http.Response`.
- `Exchange` (`exchange.go`) — both `request:` and `response:`; `Assert()`
  performs the HTTP call and asserts the response.

**Loading and templating** (`golden.go`): `Source` pairs a path with a reader.
`SourceFrom`/`Open` read a file and, when `data != nil`, run it through
`text/template` first (delimiters overridable via the `Delims` option). This is
what makes golden files "mutable" per test.

**`Meta`** (`meta.go`) is a `map[string]any` with typed, error-returning getters
(`MetaGetString`, `MetaGetInt64`, `MetaGetTime`, `MetaGetLoc`, ...). Errors wrap
the sentinel values `ErrMissing` / `ErrType` / `ErrFormat` / `ErrValue` — assert
with `errors.Is`, not string matching.

Header assertions are **subset** checks: every header in the golden file must be
present, but the actual request/response may carry more.

## Testing conventions

- Tests use `github.com/ctx42/testing` (`assert`, `check`, `tester`, `notice`) —
  **not** the standard-library `testing` assertions or testify.
- To assert that library code marks a test failed, build a spy with
  `tspy := tester.New(t)`, pass it in as the `tester.T`, then assert on the spy.
- Tests follow a `// --- Given ---` / `// --- When ---` / `// --- Then ---` block
  structure; match it in new tests.
- YAML fixtures live in `pkg/goldkit/testdata/`; resolve paths with
  `kit.AbsPath(t, "testdata/...")`.
- Production code emits assertion failures through
  `notice.New(...).Want(...).Have(...)` and `t.Error`; reuse that pattern rather
  than `fmt.Errorf` for mismatch messages.

## Go style

Before writing or editing any `.go` file, consult the project Go style
conventions (Claude Code: the `golang:style` skill), and review changes
afterward (the `golang:review` skill). The lint config (`tmp/.golangci.yml`)
enforces, among others, cyclomatic and cognitive complexity limits and forbids
`fmt.Print*`.

## Releases

`goldkit` is a standard Go module, versioned with SemVer git tags (`vX.Y.Z`);
`go get` and the module proxy resolve versions from those tags. There is no
committed version file.
