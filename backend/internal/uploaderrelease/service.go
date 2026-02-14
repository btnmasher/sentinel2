package uploaderrelease

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/config"
)

const (
	linuxAssetSuffix   = "-linux-amd64.zip"
	windowsAssetSuffix = "-windows-amd64.zip"
	macOSAssetSuffix   = "-darwin-arm64.zip"

	defaultHTTPTimeout   = 15 * time.Second
	githubAcceptHeader   = "application/vnd.github+json"
	githubAPIVersionDate = "2022-11-28"
)

type DownloadLinks struct {
	LinuxURL       string `json:"linux_url"`
	WindowsURL     string `json:"windows_url"`
	MacOSURL       string `json:"macos_url"`
	ReleasePageURL string `json:"release_page_url"`
	UpdatedAt      int64  `json:"updated_at"`
}

type githubLatestRelease struct {
	HTMLURL string               `json:"html_url"`
	Assets  []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type cacheState struct {
	links        DownloadLinks
	etag         string
	lastModified string
	nextRefresh  time.Time
}

type Service struct {
	app    *pocketbase.PocketBase
	cfg    config.Config
	client *http.Client

	mu    sync.RWMutex
	state cacheState
}

func New(app *pocketbase.PocketBase, cfg config.Config) *Service {
	s := &Service{
		app: app,
		cfg: cfg,
		client: &http.Client{
			Timeout: defaultHTTPTimeout,
		},
	}
	s.state.links.ReleasePageURL = s.latestReleasePageURL()
	return s
}

func (s *Service) Snapshot() DownloadLinks {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := s.state.links
	if snapshot.ReleasePageURL == "" {
		snapshot.ReleasePageURL = s.latestReleasePageURL()
	}
	return snapshot
}

func (s *Service) RefreshNeeded(now time.Time) bool {
	s.mu.RLock()
	nextRefresh := s.state.nextRefresh
	s.mu.RUnlock()
	return nextRefresh.IsZero() || !now.Before(nextRefresh)
}

func (s *Service) Refresh(ctx context.Context) (bool, error) {
	if !s.RefreshNeeded(time.Now()) {
		return false, nil
	}

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, s.latestReleaseAPIURL(), nil)
	if reqErr != nil {
		return false, reqErr
	}
	req.Header.Set("Accept", githubAcceptHeader)
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersionDate)
	req.Header.Set("User-Agent", s.cfg.ESIUserAgent)

	s.mu.RLock()
	etag := s.state.etag
	lastModified := s.state.lastModified
	s.mu.RUnlock()
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, respErr := s.client.Do(req)
	if respErr != nil {
		return false, respErr
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		s.applyCacheHeaders(resp, false)
		return false, nil
	case http.StatusOK:
		var payload githubLatestRelease
		if decodeErr := json.NewDecoder(resp.Body).Decode(&payload); decodeErr != nil {
			return false, decodeErr
		}
		links := s.extractLinks(payload)
		s.applyFetched(resp, links)
		return true, nil
	default:
		return false, fmt.Errorf("github latest release fetch failed: status=%d", resp.StatusCode)
	}
}

func (s *Service) extractLinks(payload githubLatestRelease) DownloadLinks {
	links := DownloadLinks{
		ReleasePageURL: strings.TrimSpace(payload.HTMLURL),
		UpdatedAt:      time.Now().Unix(),
	}
	if links.ReleasePageURL == "" {
		links.ReleasePageURL = s.latestReleasePageURL()
	}
	for _, asset := range payload.Assets {
		name := strings.TrimSpace(asset.Name)
		url := strings.TrimSpace(asset.BrowserDownloadURL)
		if name == "" || url == "" {
			continue
		}
		switch {
		case strings.HasSuffix(name, linuxAssetSuffix):
			links.LinuxURL = url
		case strings.HasSuffix(name, windowsAssetSuffix):
			links.WindowsURL = url
		case strings.HasSuffix(name, macOSAssetSuffix):
			links.MacOSURL = url
		}
	}
	return links
}

func (s *Service) applyFetched(resp *http.Response, links DownloadLinks) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.links = links
	s.state.etag = strings.TrimSpace(resp.Header.Get("ETag"))
	s.state.lastModified = strings.TrimSpace(resp.Header.Get("Last-Modified"))
	s.state.nextRefresh = parseCacheExpiry(resp.Header, time.Now())
}

func (s *Service) applyCacheHeaders(resp *http.Response, touchUpdatedAt bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if etag := strings.TrimSpace(resp.Header.Get("ETag")); etag != "" {
		s.state.etag = etag
	}
	if lastModified := strings.TrimSpace(resp.Header.Get("Last-Modified")); lastModified != "" {
		s.state.lastModified = lastModified
	}
	if touchUpdatedAt {
		s.state.links.UpdatedAt = time.Now().Unix()
	}
	s.state.nextRefresh = parseCacheExpiry(resp.Header, time.Now())
}

func (s *Service) latestReleaseAPIURL() string {
	return fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", strings.TrimSpace(s.cfg.UploaderGitHubRepo))
}

func (s *Service) latestReleasePageURL() string {
	return fmt.Sprintf("https://github.com/%s/releases/latest", strings.TrimSpace(s.cfg.UploaderGitHubRepo))
}

func parseCacheExpiry(header http.Header, now time.Time) time.Time {
	cacheControl := strings.TrimSpace(header.Get("Cache-Control"))
	if cacheControl != "" {
		for part := range strings.SplitSeq(cacheControl, ",") {
			part = strings.TrimSpace(part)
			if !strings.HasPrefix(part, "max-age=") {
				continue
			}
			secondsRaw := strings.TrimSpace(strings.TrimPrefix(part, "max-age="))
			seconds, parseErr := strconv.Atoi(secondsRaw)
			if parseErr != nil || seconds <= 0 {
				continue
			}
			return now.Add(time.Duration(seconds) * time.Second)
		}
	}

	expiresRaw := strings.TrimSpace(header.Get("Expires"))
	if expiresRaw == "" {
		return time.Time{}
	}
	expiresAt, parseErr := http.ParseTime(expiresRaw)
	if parseErr != nil {
		return time.Time{}
	}
	return expiresAt
}
