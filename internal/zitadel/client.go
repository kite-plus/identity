// Package zitadel is a small JSON client for the ZITADEL REST gateway.
package zitadel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	// BaseURL is where requests go, e.g. http://localhost:8091 or
	// http://zitadel:8080 from inside the compose network.
	BaseURL string
	// Host overrides the Host header; ZITADEL picks the instance by it.
	Host  string
	Token string
	HTTP  *http.Client
}

func New(baseURL, host, token string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Host:    host,
		Token:   strings.TrimSpace(token),
		HTTP:    &http.Client{Timeout: 30 * time.Second},
	}
}

// APIError is a non-2xx answer from the gateway.
type APIError struct {
	Status  int
	Code    int    `json:"code"`
	Message string `json:"message"`
	Method  string
	Path    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s %s: http %d: %s", e.Method, e.Path, e.Status, e.Message)
}

func IsNotFound(err error) bool {
	if e, ok := errors.AsType[*APIError](err); ok {
		return e.Status == http.StatusNotFound
	}
	return false
}

func IsAlreadyExists(err error) bool {
	if e, ok := errors.AsType[*APIError](err); ok {
		return e.Status == http.StatusConflict
	}
	return false
}

// IsNoChange reports ZITADEL's refusal of an update that changes nothing.
func IsNoChange(err error) bool {
	if e, ok := errors.AsType[*APIError](err); ok {
		m := strings.ToLower(e.Message)
		return e.Code == 9 && (strings.Contains(m, "not been changed") || strings.Contains(m, "no changes"))
	}
	return false
}

// Call sends in as JSON and decodes the answer into out. orgID, when set,
// selects the organization for management API calls.
func (c *Client) Call(ctx context.Context, method, path, orgID string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := c.newRequest(ctx, method, path, orgID, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.do(req, out)
}

// Upload posts a file as multipart form field "file".
func (c *Client) Upload(ctx context.Context, path, filename, contentType string, data []byte) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Disposition": {fmt.Sprintf(`form-data; name="file"; filename=%q`, filename)},
		"Content-Type":        {contentType},
	})
	if err != nil {
		return err
	}
	if _, err := part.Write(data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	req, err := c.newRequest(ctx, http.MethodPost, path, "", &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	return c.do(req, nil)
}

// Fetch downloads an instance URL, such as an uploaded branding asset, from
// BaseURL with the client's credentials.
func (c *Client) Fetch(ctx context.Context, rawURL string) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	req, err := c.newRequest(ctx, http.MethodGet, u.RequestURI(), "", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		return nil, &APIError{Status: resp.StatusCode, Message: string(b), Method: http.MethodGet, Path: u.Path}
	}
	return b, nil
}

func (c *Client) newRequest(ctx context.Context, method, path, orgID string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, body)
	if err != nil {
		return nil, err
	}
	if c.Host != "" {
		req.Host = c.Host
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	// Error messages are localized; IsNoChange matches the English text.
	req.Header.Set("Accept-Language", "en")
	if orgID != "" {
		req.Header.Set("x-zitadel-orgid", orgID)
	}
	return req, nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode/100 != 2 {
		e := &APIError{Status: resp.StatusCode, Method: req.Method, Path: req.URL.Path}
		if json.Unmarshal(b, e) != nil || e.Message == "" {
			e.Message = strings.TrimSpace(string(b))
		}
		return e
	}
	if out == nil || len(b) == 0 {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return fmt.Errorf("%s %s: decode: %w", req.Method, req.URL.Path, err)
	}
	return nil
}
