[![Go](https://github.com/ctx42/goldkit/actions/workflows/go.yml/badge.svg)](https://github.com/ctx42/goldkit/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ctx42/goldkit.svg)](https://pkg.go.dev/github.com/ctx42/goldkit)
[![Go Report Card](https://goreportcard.com/badge/github.com/ctx42/goldkit)](https://goreportcard.com/report/github.com/ctx42/goldkit)

# goldkit

Golden-file testing for Go — readable YAML fixtures, per-test templating, and a
whole HTTP request/response exchange asserted from a single file.

<!-- TOC -->
* [goldkit](#goldkit)
  * [Overview](#overview)
  * [Features](#features)
  * [Installation](#installation)
  * [Quickstart](#quickstart)
  * [Usage](#usage)
    * [Text and JSON bodies](#text-and-json-bodies)
    * [Templating](#templating)
    * [Custom delimiters](#custom-delimiters)
    * [Typed metadata](#typed-metadata)
    * [Multipart bodies](#multipart-bodies)
    * [HTTP requests and responses](#http-requests-and-responses)
    * [Full HTTP exchange](#full-http-exchange)
  * [Golden file format](#golden-file-format)
  * [Resources](#resources)
<!-- TOC -->

## Overview

`goldkit` compares values under test against *golden files*: expected outputs
kept beside your tests. They are **structured, commentable YAML** — not opaque
blobs — so a reviewer sees exactly what changed in a diff, and a test can render
the fixture as a Go template when it needs per-case data.

Its standout capability is HTTP: a single golden file can describe a **whole
exchange** — the request, the expected response, even multipart bodies — and
`goldkit` replays the request against a server and asserts the response for you.
It builds on [ctx42/testing](https://github.com/ctx42/testing) for assertions
and pairs with [ctx42/testkit](https://github.com/ctx42/testkit)'s `httpkit`
for driving test servers.

## Features

- **One-file HTTP exchanges** — define a request and its expected response
  (multipart included) in one YAML file and assert the round trip.
- **Readable YAML fixtures** — structured, commentable golden files that review
  cleanly in a pull request.
- **Templating** — render a fixture with per-test data via Go `text/template`;
  delimiters are configurable.
- **Semantic JSON equality** — JSON bodies match on data, not byte layout, so
  formatting and key order don't break a test.
- **Multipart bodies** — build and assert `multipart/form-data` with a fixed
  boundary, or get a ready-to-send `*http.Request`.
- **Typed metadata** — attach free-form `meta` to a fixture and read it back
  with typed, error-returning getters.

## Installation

```bash
go get github.com/ctx42/goldkit
```

```go
import "github.com/ctx42/goldkit/pkg/goldkit"
```

## Quickstart

Keep the expected output in a golden file next to your test:

```yaml
# testdata/golden.yml
bodyType: text
body: |
  Hello, Alice!
```

Assert a value against it:

```go
func TestGreeting(t *testing.T) {
	gld := goldkit.Create(t, "testdata/golden.yml", nil)
	gld.Assert([]byte("Hello, Alice!\n"))
}
```

> [!NOTE]
> `Create` and `Assert` take your test's `*testing.T`, so that flow lives in a
> test function. The fenced examples further down are runnable Go `Example`
> functions kept in sync with the package tests.

## Usage

### Text and JSON bodies

A golden file declares a `bodyType`. Text bodies compare byte-for-byte; JSON
bodies compare by **semantic equality**, so indentation and key order don't
matter.

```yaml
# testdata/base_json.yml
bodyType: json
body: |-
  {"key2": "val2"}
```

Against that fixture, both of these pass:

```go
gld := goldkit.Create(t, "testdata/base_json.yml", nil)
gld.Assert([]byte(`{"key2":"val2"}`))              // compact
gld.Assert([]byte("{\n  \"key2\": \"val2\"\n}"))   // pretty — same data
```

### Templating

Pass a data object and the golden file is rendered as a Go template before use,
so one fixture serves many cases:

```yaml
# testdata/eg_greeting.tpl.yml
body: |
  Hello, {{ .Name }}!
```

<!-- gmdoceg:pkg/goldkit/ExampleSourceFrom -->
```go
m := map[string]string{"Name": "alice"}
src, err := goldkit.SourceFrom("testdata/eg_greeting.tpl.yml", m)
if err != nil {
	panic(err)
}

body, _ := io.ReadAll(src)
fmt.Print(string(body))
// Output:
// body: |
//   Hello, alice!
```

### Custom delimiters

When the golden content itself contains `{{`/`}}`, switch the template
delimiters with `Delims`:

<!-- gmdoceg:pkg/goldkit/ExampleDelims -->
```go
data := map[string]string{"key1": "val1"}
src, err := goldkit.SourceFrom(
	"testdata/golden_custom_delim.tpl.yml",
	data,
	goldkit.Delims("[[", "]]"),
)
if err != nil {
	panic(err)
}

body, _ := io.ReadAll(src)
fmt.Print(string(body))
// Output:
// meta:
//   key1: val1
```

### Typed metadata

A golden file — or any `Meta` map — carries free-form key/values you read back
with typed, error-returning getters:

<!-- gmdoceg:pkg/goldkit/ExampleMeta -->
```go
m := goldkit.Meta{"user": "alice", "attempts": 3}

user, _ := m.MetaGetString("user")
attempts, _ := m.MetaGetInt("attempts")

fmt.Printf("%s made %d attempts\n", user, attempts)
// Output:
// alice made 3 attempts
```

Richer scalar types are parsed and validated too — RFC 3339 timestamps, floats,
ints, durations, and locations:

<!-- gmdoceg:pkg/goldkit/ExampleMeta_typedScalars -->
```go
m := goldkit.Meta{"when": "2000-01-02T03:04:05Z", "ratio": 12.5}

when, _ := m.MetaGetTime("when")
ratio, _ := m.MetaGetFloat64("ratio")

fmt.Println(when.Format("2006-01-02"))
fmt.Println(ratio)
// Output:
// 2000-01-02
// 12.5
```

### Multipart bodies

Build a `multipart/form-data` body with a fixed boundary:

<!-- gmdoceg:pkg/goldkit/ExampleMultiPart -->
```go
mp := goldkit.NewMultipart()
if err := mp.SetBoundary("example-boundary"); err != nil {
	panic(err)
}
if err := mp.AddField("greeting", "hello"); err != nil {
	panic(err)
}

h := make(http.Header)
mp.SetContentTypeHeader(h)

fmt.Println(h.Get("Content-Type"))
// Print the wire body one line at a time so the CRLF framing is visible.
for _, line := range bytes.Split(mp.Body(), []byte("\r\n")) {
	fmt.Printf("%q\n", line)
}
// Output:
// multipart/form-data; boundary=example-boundary
// "--example-boundary"
// "Content-Disposition: form-data; name=\"greeting\""
// ""
// "hello"
// "--example-boundary--"
// ""
```

Or get a ready-to-send `*http.Request` with the body and `Content-Type` already
set:

<!-- gmdoceg:pkg/goldkit/ExampleMultiPart_request -->
```go
mp := goldkit.NewMultipart()
if err := mp.SetBoundary("example-boundary"); err != nil {
	panic(err)
}
if err := mp.AddField("greeting", "hello"); err != nil {
	panic(err)
}

req, err := mp.Request(http.MethodPost, "/upload")
if err != nil {
	panic(err)
}

fmt.Println(req.Method, req.URL.Path)
fmt.Println(req.Header.Get("Content-Type"))
// Output:
// POST /upload
// multipart/form-data; boundary=example-boundary
```

### HTTP requests and responses

Describe a request or response entirely in YAML, then assert an actual
`*http.Request` or `*http.Response` against it:

```yaml
# testdata/request_full.yml
request:
  method: POST
  path: /some/path
  query: key0=val0&key1=val1
  headers:
    - 'Authorization: Bearer token'
  bodyType: text
  body: |
    abc
```

```go
func TestOutboundRequest(t *testing.T) {
	_, src := goldkit.Open(t, "testdata/request_full.yml", nil)
	want := goldkit.NewRequest(t, src)

	got := buildRequest() // the *http.Request your code produces
	want.Assert(got)
}
```

### Full HTTP exchange

The headline: one file holds both the request and the expected response.
`goldkit` replays the request against a server and asserts the response in a
single call.

```yaml
# testdata/exchange.yml (abbreviated)
request:
  scheme: http
  host: {{ .host }}
  method: POST
  path: /some/path
  query: key0=val0&key1=val1
  headers:
    - 'Authorization: Bearer token'
  bodyType: json
  body: |
    {"key2": "val2"}
response:
  statusCode: 200
  headers:
    - 'Content-Type: application/json'
  bodyType: json
  body: |
    {"success": true}
```

```go
func TestExchange(t *testing.T) {
	// A recording test server that returns the canned response.
	srv := httpkit.NewServer(t)
	srv.Rsp(http.StatusOK, []byte(`{"success": true}`)).
		Header("Content-Type", "application/json")

	// The exchange file targets that server via its templated host.
	u, _ := url.Parse(srv.URL())
	data := goldkit.Meta{}.MetaSet("host", u.Host)
	src, _ := goldkit.SourceFrom("testdata/exchange.yml", data)

	// Replay the request and assert the response in one call.
	req, res := goldkit.NewExchange(t, src).Assert()
	_, _ = req, res
}
```

> [!NOTE]
> This example drives a live test server on a random port, so it is illustrative
> rather than a deterministic Go `Example`. The exchange file's host is
> templated (`{{ .host }}`) and filled in per run.

## Golden file format

A value-under-test golden file:

| Key        | Meaning                                                    |
|------------|------------------------------------------------------------|
| `bodyType` | `text` (default), `json`, `multipart`, or `none`.          |
| `body`     | Expected body; rendered as a template when data is passed. |
| `meta`     | Free-form key/values, read via the typed `Meta` getters.   |

HTTP golden files wrap fields under `request:` and/or `response:` — `method`,
`path`, `query`, `headers`, `statusCode`, plus the same `bodyType`, `body`, and
`meta` keys. A file with both a `request:` and a `response:` is an exchange.

## Resources

- [API reference](https://pkg.go.dev/github.com/ctx42/goldkit) — full package docs.
- [ctx42/testing](https://github.com/ctx42/testing) — the assertion library goldkit builds on.
- [ctx42/testkit](https://github.com/ctx42/testkit) — `httpkit` test servers and clients.
