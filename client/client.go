package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/ATMackay/checkout/model"
	"github.com/ATMackay/checkout/server"
)

var ErrMethodNotAllowed = errors.New("method not allowed")

type Client struct {
	baseURL string
	c       *http.Client
	mu      sync.Mutex
	headers http.Header
}

// New returns a new checkout server http client.
func New(url string) *Client {
	return &Client{
		baseURL: url,
		c:       new(http.Client),
		mu:      sync.Mutex{},
		headers: makeDefaultHeaders(),
	}
}

func makeDefaultHeaders() http.Header {
	h := make(http.Header)
	h.Set("Content-Type", "application/json")
	return h
}

func (c *Client) AddAuthorizationHeader(psswd string) {
	c.headers.Set("X-Auth-Password", psswd)
}

func (client *Client) Status(ctx context.Context) (*model.StatusResponse, error) {
	var status model.StatusResponse
	if err := client.executeRequest(ctx, &status, http.MethodGet, server.StatusEndPnt, nil); err != nil {
		return nil, err
	}
	return &status, nil
}

func (client *Client) Health(ctx context.Context) (*model.HealthResponse, error) {
	var health model.HealthResponse
	if err := client.executeRequest(ctx, &health, http.MethodGet, server.HealthEndPnt, nil); err != nil {
		return nil, err
	}
	return &health, nil
}

func (client *Client) AddItems(ctx context.Context, addItemReq *model.AddItemsRequest) error {
	if err := client.executeRequest(ctx, nil, http.MethodPost, server.ItemsEndPnt, addItemReq); err != nil {
		return err
	}
	return nil
}

func (client *Client) GetItemPrice(ctx context.Context, key string) (*model.PriceResponse, error) {
	var itPriceResp model.PriceResponse
	if err := client.executeRequest(ctx, &itPriceResp, http.MethodGet, fmt.Sprintf("%s/%s", server.ItemPriceEndPnt, key), nil); err != nil {
		return nil, err
	}
	return &itPriceResp, nil
}

func (client *Client) GetItemsPrice(ctx context.Context, itemsPriceReq *model.ItemsPriceRequest) (*model.PriceResponse, error) {
	var itPriceResp model.PriceResponse
	if err := client.executeRequest(ctx, &itPriceResp, http.MethodPost, server.ItemPriceEndPnt, itemsPriceReq); err != nil {
		return nil, err
	}
	return &itPriceResp, nil
}

func (client *Client) PurchaseItems(ctx context.Context, itemsPriceReq *model.PurchaseItemsRequest) (*model.PurchaseItemsResponse, error) {
	var itPurchaseResp model.PurchaseItemsResponse
	if err := client.executeRequest(ctx, &itPurchaseResp, http.MethodPost, server.ItemPurchaseEndPnt, itemsPriceReq); err != nil {
		return nil, err
	}
	return &itPurchaseResp, nil
}

func (client *Client) GetOrders(ctx context.Context) (*model.Orders, error) {
	var orders model.Orders
	if err := client.executeRequest(ctx, &orders, http.MethodGet, server.OrdersEndPnt, nil); err != nil {
		return nil, err
	}
	return &orders, nil
}

func (client *Client) executeRequest(ctx context.Context, result any, method, path string, body any) (err error) {

	op := &requestOp{
		path:   path,
		method: method,
		msg:    body,
		resp:   make(chan *jsonResult, 1),
	}
	if err := client.sendHTTP(ctx, op, result); err != nil {
		return err
	}

	jsonRes, err := op.wait(ctx)
	if err != nil {
		return err
	}
	if jsonRes.errMsg != nil {
		return fmt.Errorf("%v", jsonRes.errMsg.Error)
	}

	return nil
}

func (client *Client) sendHTTP(ctx context.Context, op *requestOp, result any) error {

	respBody, status, err := client.doRequest(ctx, op.method, op.path, op.msg)
	if err != nil {
		return err
	}

	defer respBody.Close()

	// await response
	var res = &jsonResult{
		result: result,
	}

	// process resp or error
	if status >= http.StatusBadRequest {
		if status == http.StatusMethodNotAllowed {
			return fmt.Errorf("method: '%v', path: '%v' %w", op.method, op.path, ErrMethodNotAllowed)
		}
		errMsg := server.JSONError{}
		if err := json.NewDecoder(respBody).Decode(&errMsg); err != nil {
			return err
		}
		res.errMsg = &errMsg
	} else if result != nil {
		if err := json.NewDecoder(respBody).Decode(&result); err != nil {
			return err
		}
	}

	op.resp <- res

	return nil
}

func (client *Client) doRequest(ctx context.Context, method, path string, msg any) (io.ReadCloser, int, error) {
	// Serialize JSON-encoded method
	var body []byte
	var err error
	if msg != nil {
		body, err = json.Marshal(msg)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, client.baseURL+path, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	req.ContentLength = int64(len(body))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }

	// set headers
	client.mu.Lock()
	req.Header = client.headers.Clone()
	client.mu.Unlock()
	setHeaders(req.Header, headersFromContext(ctx))

	// do request
	resp, err := client.c.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return resp.Body, resp.StatusCode, nil
}

type jsonResult struct {
	result any
	errMsg *server.JSONError
}

type requestOp struct {
	path   string
	method string
	msg    any
	resp   chan *jsonResult
}

func (op *requestOp) wait(ctx context.Context) (*jsonResult, error) {
	select {
	case <-ctx.Done():
		// Send the timeout error
		return nil, ctx.Err()
	case resp := <-op.resp:
		return resp, nil
	}
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
