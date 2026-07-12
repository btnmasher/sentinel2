package esi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DoTestAuthProxyRequest sends an ESI request through the TestAuth proxy.
func DoTestAuthProxyRequest(ctx context.Context, baseURL, accessToken, characterID, method, path string, body io.Reader, extraHeaders http.Header) (*http.Response, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("missing testauth base url")
	}

	base, baseErr := url.Parse(strings.TrimRight(baseURL, "/") + "/oauth/api/esi-proxy/")
	if baseErr != nil {
		return nil, baseErr
	}

	rel, relErr := url.Parse(strings.TrimLeft(path, "/"))
	if relErr != nil {
		return nil, relErr
	}

	proxyURL := base.ResolveReference(rel)
	query := proxyURL.Query()
	if strings.TrimSpace(characterID) != "" {
		query.Set("character_id", characterID)
	}
	proxyURL.RawQuery = query.Encode()

	reqBody := body
	if reqBody == nil && (method == http.MethodGet || method == http.MethodHead) {
		reqBody = http.NoBody
	}

	req, reqErr := http.NewRequestWithContext(ctx, method, proxyURL.String(), reqBody)
	if reqErr != nil {
		return nil, reqErr
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")
	for key, values := range extraHeaders {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{Timeout: defaultESIProxyTimeout}
	return client.Do(req)
}
