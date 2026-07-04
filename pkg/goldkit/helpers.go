package goldkit

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"

	"github.com/ctx42/testing/pkg/tester"
	"gopkg.in/yaml.v3"
)

// ErrInvBodyType represents an unsupported request body type error.
var ErrInvBodyType = errors.New("invalid body type")

// parseBody takes a path to a golden file, its body node and its type and
// parses it. Returns YAML parsing error or [ErrInvBodyType] if typ is unknown.
func parseBody(pth string, body yaml.Node, typ string) (Body, error) {
	switch typ {
	case JSON:
		return jsonBody([]byte(body.Value)), nil

	case Text:
		return textBody(body.Value), nil

	case Multipart:
		mp := newMpBody(filepath.Dir(pth))
		if err := body.Decode(mp); err != nil {
			return nil, fmt.Errorf("decode multipart body: %w", err)
		}
		if err := mp.parse(); err != nil {
			return nil, err
		}
		return mp, nil

	case None:
		return noneBody(body.Value)

	default:
		return nil, ErrInvBodyType
	}
}

// cloneReader reads all bytes from rc and returns it as a slice and
// [io.ReadCloser] with the same data, so it can be used, for example, to
// "reset" body of a [http.Request] or [http.Response] instance.
func cloneReader(t tester.T, rc io.ReadCloser) ([]byte, io.ReadCloser) {
	t.Helper()
	buf := &bytes.Buffer{}
	tee := io.TeeReader(rc, buf)
	data, err := io.ReadAll(tee)
	if err != nil {
		t.Error(err)
		return nil, nil
	}
	_ = rc.Close()
	return data, io.NopCloser(buf)
}

// lines2Headers creates [http.Header] from header lines.
func lines2Headers(lines ...string) (http.Header, error) {
	if len(lines) == 0 {
		return make(http.Header), nil
	}
	rdr := strings.NewReader(strings.Join(lines, "\r\n") + "\r\n\r\n")
	tp := textproto.NewReader(bufio.NewReader(rdr))
	hs, err := tp.ReadMIMEHeader()
	if err != nil {
		return nil, err
	}
	return http.Header(hs), nil
}

// findBoundary is a really crud way of finding boundary in a multipart body.
func findBoundary(body []byte) (string, error) {
	str, err := bytes.NewBuffer(body).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("find boundary: %w", err)
	}
	if !strings.HasPrefix(str, "--") || !strings.HasSuffix(str, "\r\n") {
		return "", errors.New("find boundary: invalid multipart body")
	}
	str = strings.TrimSuffix(str, "\r\n")
	return strings.TrimPrefix(str, "--"), nil
}
