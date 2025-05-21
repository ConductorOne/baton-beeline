package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

type tokenSource struct {
	authServerURL string
	clientID      string
	clientSecret  string
	baseAPIURL    *url.URL
	httpClient    *http.Client
}

type token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	if ts.clientID == "" || ts.clientSecret == "" || ts.baseAPIURL == nil {
		return nil, fmt.Errorf("invalid token source configuration: clientID, clientSecret, and baseAPIURL are required")
	}

	payload := url.Values{}
	payload.Set("client_id", ts.clientID)
	payload.Set("client_secret", ts.clientSecret)
	payload.Set("grant_type", "client_credentials")

	baseURL := *ts.baseAPIURL
	payload.Set("audience", baseURL.String())

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.authServerURL, bytes.NewBufferString(payload.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	token := &token{}
	err = json.Unmarshal(body, token)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling token response: %w", err)
	}

	if token.AccessToken == "" {
		return nil, fmt.Errorf("received empty access token")
	}

	if token.ExpiresIn <= 0 {
		// Set a default expiration of 1 hour if not provided
		token.ExpiresIn = 3600
	}

	return &oauth2.Token{
		AccessToken: token.AccessToken,
		Expiry:      time.Now().Add(time.Duration(token.ExpiresIn) * time.Second),
		TokenType:   token.TokenType,
	}, nil
}

func newTokenSource(authServerURL, clientID, clientSecret string, baseAPIURL *url.URL, httpClient *http.Client) (oauth2.TokenSource, error) {
	return oauth2.ReuseTokenSource(nil, &tokenSource{
		authServerURL: authServerURL,
		clientID:      clientID,
		clientSecret:  clientSecret,
		baseAPIURL:    baseAPIURL,
		httpClient:    httpClient,
	}), nil
}
