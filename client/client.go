package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ATMackay/checkout/model"
	"github.com/ATMackay/checkout/server"
)

var ErrMethodNotAllowed = errors.New("method not allowed")

type Client struct {
	base *url.URL
	http *http.Client
	mu   sync.RWMutex
	hdr  http.Header
}

type Option func(*Client)

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.http = hc }
}

func New(baseURL string, opts ...Option) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid baseURL: %w", err)
	}
	c := &Client{
		base: u,
		http: &http.Client{Timeout: 10 * time.Second},
		hdr:  http.Header{"Accept": []string{"application/json"}},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

func (c *Client) AddAuthorizationHeader(psswd string) {
	c.mu.Lock()
	c.hdr.Set("X-Auth-Password", psswd)
	c.mu.Unlock()
}

type HTTPError struct {
	Status int
	Body   []byte
	JSON   *server.JSONError
}

func (e *HTTPError) Error() string {
	if e.JSON != nil && e.JSON.Error != "" {
		return fmt.Sprintf("http %d: %s", e.Status, e.JSON.Error)
	}
	return fmt.Sprintf("http %d: %s", e.Status, string(e.Body))
}

func (c *Client) executeJSONRequest(ctx context.Context, method, path string, in any, out any) error {
	// Build URL
	u := *c.base
	var err error
	u.Path, err = url.JoinPath(u.Path, path)
	if err != nil {
		return err
	}

	// Encode body if present
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return err
	}

	// Set headers (clone defaults, then overlay ctx headers)
	c.mu.RLock()
	req.Header = c.hdr.Clone()
	c.mu.RUnlock()
	// Set default content type header
	req.Header.Set("Content-Type", "application/json")
	// Add additional header from context
	setHeaders(req.Header, headersFromContext(ctx))

	// Do request
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		he := &HTTPError{Status: resp.StatusCode, Body: b}
		var je server.JSONError
		if json.Unmarshal(b, &je) == nil && (je.Error != "") {
			he.JSON = &je
		}
		if resp.StatusCode == http.StatusMethodNotAllowed {
			return fmt.Errorf("method %s %s: %w", method, path, ErrMethodNotAllowed)
		}
		return he
	}
	if out != nil {
		dec := json.NewDecoder(bytes.NewReader(b))
		if err := dec.Decode(out); err != nil {
			return err
		}
	}
	return nil
}

func (client *Client) Status(ctx context.Context) (*model.StatusResponse, error) {
	var status model.StatusResponse
	if err := client.executeJSONRequest(ctx, http.MethodGet, server.StatusEndPnt, nil, &status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (client *Client) Health(ctx context.Context) (*model.HealthResponse, error) {
	var health model.HealthResponse
	if err := client.executeJSONRequest(ctx, http.MethodGet, server.HealthEndPnt, nil, &health); err != nil {
		return nil, err
	}
	return &health, nil
}

func (client *Client) AddItems(ctx context.Context, addItemReq *model.AddItemsRequest) error {
	if err := client.executeJSONRequest(ctx, http.MethodPost, server.ItemsEndPnt, addItemReq, nil); err != nil {
		return err
	}
	return nil
}

func (client *Client) GetItemPrice(ctx context.Context, key string) (*model.PriceResponse, error) {
	var itPriceResp model.PriceResponse
	if err := client.executeJSONRequest(ctx, http.MethodGet, fmt.Sprintf("%s/%s", server.ItemPriceEndPnt, key), nil, &itPriceResp); err != nil {
		return nil, err
	}
	return &itPriceResp, nil
}

func (client *Client) GetItemsPrice(ctx context.Context, itemsPriceReq *model.ItemsPriceRequest) (*model.PriceResponse, error) {
	var itPriceResp model.PriceResponse
	if err := client.executeJSONRequest(ctx, http.MethodPost, server.ItemPriceEndPnt, itemsPriceReq, &itPriceResp); err != nil {
		return nil, err
	}
	return &itPriceResp, nil
}

func (client *Client) PurchaseItems(ctx context.Context, itemsPriceReq *model.PurchaseItemsRequest) (*model.PurchaseItemsResponse, error) {
	var itPurchaseResp model.PurchaseItemsResponse
	if err := client.executeJSONRequest(ctx, http.MethodPost, server.ItemPurchaseEndPnt, itemsPriceReq, &itPurchaseResp); err != nil {
		return nil, err
	}
	return &itPurchaseResp, nil
}

func (client *Client) GetOrders(ctx context.Context) (*model.Orders, error) {
	var orders model.Orders
	if err := client.executeJSONRequest(ctx, http.MethodGet, server.OrdersEndPnt, nil, &orders); err != nil {
		return nil, err
	}
	return &orders, nil
}

type mdHeaderKey struct{}

// headersFromContext is used to extract http.Header from context.
func headersFromContext(ctx context.Context) http.Header {
	source, _ := ctx.Value(mdHeaderKey{}).(http.Header)
	return source
}

// setHeaders sets all headers from src in dst.
func setHeaders(dst http.Header, src http.Header) http.Header {
	for key, values := range src {
		dst[http.CanonicalHeaderKey(key)] = values
	}
	return dst
}
