package web

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

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
		proxy.ServeHTTP(c.Response, c.Request)
		return nil
	})

	return nil
}
