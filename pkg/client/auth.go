package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

//nolint:gosec // This constant is the token endpoint URL, not a secret token.
const tokenURL = "https://integrations.auth.beeline.com/oauth/token"

type tokenSource struct {
	clientID     string
	clientSecret string
	baseAPIURL   *url.URL
	httpClient   *http.Client
}

type token struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (ts *tokenSource) Token() (*oauth2.Token, error) {
	payload := url.Values{}
	payload.Set("client_id", ts.clientID)
	payload.Set("client_secret", ts.clientSecret)
	payload.Set("audience", ts.baseAPIURL.String()) // audience = the base URL for the API you are using.
	payload.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, tokenURL, bytes.NewBufferString(payload.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	token := &token{}
	err = json.Unmarshal(body, token)
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: token.AccessToken,
		Expiry:      time.Unix(int64(token.ExpiresIn), 0),
		TokenType:   token.TokenType,
	}, nil
}

func newTokenSource(clientID, clientSecret string, baseAPIURL *url.URL, httpClient *http.Client) (oauth2.TokenSource, error) {
	return oauth2.ReuseTokenSource(nil, &tokenSource{
		clientID:     clientID,
		clientSecret: clientSecret,
		baseAPIURL:   baseAPIURL,
		httpClient:   httpClient,
	}), nil
}
