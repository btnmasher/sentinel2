package web

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

const cspNoncePlaceholder = "__SENTINEL2_CSP_NONCE__"
const cspNonceContextKey = "csp_nonce"

func MountFrontendProxy(r *router.Router[*core.RequestEvent], target string) error {
	if target == "" {
		return nil
	}

	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "http://" + target
	}
	proxyURL, err := url.Parse(target)
	if err != nil {
		return err
	}
	proxy := httputil.NewSingleHostReverseProxy(proxyURL)

	skipPrefixes := []string{
		"/api",
		"/_",
	}

	r.BindFunc(func(c *core.RequestEvent) error {
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			return c.Next()
		}
		path := c.Request.URL.Path
		for _, prefix := range skipPrefixes {
			if path == prefix || strings.HasPrefix(path, prefix+"/") {
				return c.Next()
			}
		}

		c.Request.Header.Set("Accept-Encoding", "identity")

		nonce, _ := c.Get(cspNonceContextKey).(string)
		if nonce != "" {
			c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), cspNonceContextKeyType{}, nonce))
		}

		serveProxy(proxy, c.Response, c.Request)
		return nil
	})

	return nil
}

type cspNonceContextKeyType struct{}

func serveProxy(proxy *httputil.ReverseProxy, w http.ResponseWriter, req *http.Request) {
	if proxy == nil {
		return
	}

	clone := *proxy
	clone.ModifyResponse = func(resp *http.Response) error {
		if resp == nil || resp.Request == nil || resp.Body == nil {
			return nil
		}

		nonce, _ := resp.Request.Context().Value(cspNonceContextKeyType{}).(string)
		if nonce == "" {
			return nil
		}

		contentType := strings.ToLower(resp.Header.Get("Content-Type"))
		if !strings.Contains(contentType, "text/html") {
			return nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		rewritten := bytes.ReplaceAll(body, []byte(cspNoncePlaceholder), []byte(nonce))
		resp.Body = io.NopCloser(bytes.NewReader(rewritten))
		resp.ContentLength = int64(len(rewritten))
		resp.Header.Set("Content-Length", strconv.Itoa(len(rewritten)))
		resp.Header.Del("Content-Encoding")

		return nil
	}

	clone.ServeHTTP(w, req)
}
