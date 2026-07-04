package goldkit_test

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/ctx42/goldkit/pkg/goldkit"
)

// SourceFrom loads a golden file and, when given data, renders it as a Go
// template before use, so a single fixture can serve many test cases.
func ExampleSourceFrom() {
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
}

// Delims renders a fixture that uses custom template action delimiters, so
// golden content that itself contains "{{"/"}}" stays unambiguous.
func ExampleDelims() {
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
}

// Meta gives typed, error-checked access to the free-form metadata carried by
// a golden file.
func ExampleMeta() {
	m := goldkit.Meta{"user": "alice", "attempts": 3}

	user, _ := m.MetaGetString("user")
	attempts, _ := m.MetaGetInt("attempts")

	fmt.Printf("%s made %d attempts\n", user, attempts)
	// Output:
	// alice made 3 attempts
}

// Meta parses richer scalar types too — RFC 3339 timestamps and floats — each
// with an error return you can check.
func ExampleMeta_typedScalars() {
	m := goldkit.Meta{"when": "2000-01-02T03:04:05Z", "ratio": 12.5}

	when, _ := m.MetaGetTime("when")
	ratio, _ := m.MetaGetFloat64("ratio")

	fmt.Println(when.Format("2006-01-02"))
	fmt.Println(ratio)
	// Output:
	// 2000-01-02
	// 12.5
}

// MultiPart builds a MIME multipart body with a fixed boundary, ready to be
// used as an HTTP request body.
func ExampleMultiPart() {
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
}

// MultiPart can hand you a ready-to-send *http.Request with the multipart body
// and Content-Type header already set.
func ExampleMultiPart_request() {
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
}
