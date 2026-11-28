package onlineconf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const maxResponseBytes = 4 << 20

type Client struct {
	baseURL *url.URL
	http    *http.Client
}

func NewClient(rawURL string) (*Client, error) {
	base, err := url.Parse(rawURL)
	if err != nil || base.Hostname() == "" || (base.Scheme != "http" && base.Scheme != "https") ||
		base.User != nil || (base.Path != "" && base.Path != "/") || base.ForceQuery || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("ONLINECONF_URL должен быть HTTP(S)-адресом сервиса без пути и параметров")
	}
	return &Client{
		baseURL: base,
		http: &http.Client{
			Timeout: 3 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (c *Client) ListEntries(ctx context.Context, query url.Values) (int, json.RawMessage, error) {
	return c.request(ctx, http.MethodGet, "/internal/v1/entries", query.Encode(), nil, "")
}

func (c *Client) SaveEntry(ctx context.Context, scope, key string, body io.Reader, actor string) (int, json.RawMessage, error) {
	if !ValidName(scope) || !ValidName(key) {
		return 0, nil, fmt.Errorf("неверный scope или key")
	}
	return c.request(ctx, http.MethodPut, "/internal/v1/entries/"+scope+"/"+key, "", body, actor)
}

func (c *Client) request(ctx context.Context, method, path, query string, body io.Reader, actor string) (int, json.RawMessage, error) {
	target := *c.baseURL
	target.Path, target.RawPath, target.RawQuery = path, "", query
	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-OnlineConf-Actor", actor)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusBadRequest, http.StatusConflict:
	default:
		return 0, nil, fmt.Errorf("OnlineConf: неожиданный HTTP-статус %d", resp.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return 0, nil, err
	}
	if len(payload) > maxResponseBytes || !json.Valid(payload) {
		return 0, nil, fmt.Errorf("OnlineConf: неверный размер или формат ответа")
	}
	return resp.StatusCode, payload, nil
}
