package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

// get performs a GET request to the API.
func (c *Client) get(
	ctx context.Context,
	url *url.URL,
	target interface{},
) (
	*v2.RateLimitDescription,
	error,
) {
	// Get current token
	token, err := c.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return c.doRequest(
		ctx,
		http.MethodGet,
		url,
		target,
		uhttp.WithBearerToken(token.AccessToken),
	)
}

// post performs a POST request to the API.
func (c *Client) post(
	ctx context.Context,
	url *url.URL,
	target interface{},
	payload map[string]interface{},
) (
	*v2.RateLimitDescription,
	error,
) {
	// Get current token
	token, err := c.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	return c.doRequest(
		ctx,
		http.MethodPost,
		url,
		target,
		uhttp.WithBearerToken(token.AccessToken),
		uhttp.WithJSONBody(payload),
	)
}

func (c *Client) doRequest(
	ctx context.Context,
	method string,
	url *url.URL,
	target interface{},
	options ...uhttp.RequestOption,
) (
	*v2.RateLimitDescription,
	error,
) {
	logger := ctxzap.Extract(ctx)
	logger.Debug(
		"making request",
		zap.String("method", method),
		zap.String("url", url.String()),
	)

	options = append(
		options,
		uhttp.WithAcceptJSONHeader(),
		uhttp.WithContentTypeJSONHeader(),
	)

	request, err := c.httpClient.NewRequest(
		ctx,
		method,
		url,
		options...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	var ratelimitData v2.RateLimitDescription
	response, err := c.httpClient.Do(
		request,
		uhttp.WithRatelimitData(&ratelimitData),
		uhttp.WithErrorResponse(&ErrorResponse{}),
	)
	if err != nil {
		return &ratelimitData, fmt.Errorf("request failed: %w", err)
	}
	defer response.Body.Close()

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return &ratelimitData, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log response details for debugging
	logger.Debug("received response",
		zap.Int("status_code", response.StatusCode),
		zap.String("content_type", response.Header.Get("Content-Type")),
		zap.String("body", string(bodyBytes)),
	)

	if response.StatusCode != http.StatusOK {
		return &ratelimitData, fmt.Errorf("request failed with status %d: %s", response.StatusCode, string(bodyBytes))
	}

	if len(bodyBytes) == 0 {
		// If target is nil, we expect no response body
		if target == nil {
			return &ratelimitData, nil
		}
		return nil, fmt.Errorf("server returned empty response")
	}

	if err := json.Unmarshal(bodyBytes, &target); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &ratelimitData, nil
}
