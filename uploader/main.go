package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/fsnotify/fsnotify"
	flags "github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
)

type Options struct {
	BaseURL string `long:"base-url" env:"SENTINEL_BASE_URL" description:"Sentinel base URL (e.g. https://intel.example.com)" required:"true"`
	Token   string `long:"token" env:"SENTINEL_TOKEN" description:"Uploader token" required:"true"`
	LogFile string `long:"log-file" env:"SENTINEL_LOG_FILE" description:"EVE chat log file to watch"`
	LogDir  string `long:"log-dir" env:"SENTINEL_LOG_DIR" description:"Directory containing EVE chat logs"`
	Debug   bool   `long:"debug" env:"SENTINEL_DEBUG" description:"Enable verbose debug output"`
}

var reportPattern = regexp.MustCompile(`^\[ ([0-9]{4}\.[0-9]{2}\.[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}) \] ([^>]+) > (.*)$`)

type submitPayload struct {
	Text      string `json:"text"`
	ChannelID string `json:"channel_id"`
}

type channelConfig struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type uploaderConfig struct {
	Channels []channelConfig `json:"channels"`
}

type logSelection struct {
	Path    string
	Channel channelConfig
}

var (
	debugEnabled bool
	logger       = newUploaderLogger(false)
	BuildVersion = "dev"
)

func main() {
	_ = godotenv.Load()

	opts := Options{}
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}
	debugEnabled = opts.Debug
	logger = newUploaderLogger(opts.Debug)
	logger.Info("starting uploader", field("version", BuildVersion))

	apiBaseURL, err := buildAPIBaseURL(opts.BaseURL)
	if err != nil {
		logger.Error("invalid base URL", field("error", err))
		os.Exit(1)
	}
	configURL := apiBaseURL + "/uploader/config"
	submitURL := apiBaseURL + "/uploader/submit"
	debugf("api base URL: %s", apiBaseURL)

	if opts.LogDir == "" && opts.LogFile == "" {
		opts.LogDir = defaultLogDir()
		if opts.LogDir == "" {
			logger.Warn("no default log directory found; set --log-dir or --log-file")
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	channels, err := fetchChannels(client, configURL, opts.Token)
	if err != nil {
		logger.Error("failed to fetch channels", field("error", err))
		os.Exit(1)
	}
	if len(channels) == 0 {
		logger.Error("no channels configured")
		os.Exit(1)
	}

	var selected logSelection
	if opts.LogFile != "" {
		channel, ok := resolveChannelForPath(opts.LogFile, channels)
		if !ok || channel.ID == "" {
			logger.Error("failed to map log file to configured channel", field("path", opts.LogFile))
			os.Exit(1)
		}
		selected = logSelection{Path: opts.LogFile, Channel: channel}
	} else if opts.LogDir != "" {
		latest, ok := findLatestLog(opts.LogDir, channels)
		if ok {
			selected = latest
		}
	}
	if selected.Path == "" {
		logger.Error("missing log file")
		os.Exit(1)
	}
	if selected.Channel.ID == "" {
		logger.Error("selected log file does not map to a channel id", field("path", selected.Path), field("channel", selected.Channel.Name))
		os.Exit(1)
	}
	logger.Info(
		"selected log file",
		field("path", selected.Path),
		field("channel", selected.Channel.Name),
		field("channel_id", selected.Channel.ID),
		field("channels", len(channels)),
	)

	cutoff := time.Now().Add(-1 * time.Minute)
	if err := sendExistingLines(client, submitURL, opts.Token, selected.Channel.ID, selected.Path, cutoff); err != nil {
		logger.Warn("failed to read recent logs", field("error", err))
	}

	tailer := &Tailer{Path: selected.Path}
	_ = tailer.Prime()

	logger.Info("watching log file", field("path", selected.Path), field("channel", selected.Channel.Name))

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("failed to initialize fsnotify watcher", field("error", err))
		os.Exit(1)
	}
	defer watcher.Close()

	watchDir := filepath.Dir(selected.Path)
	if opts.LogDir != "" {
		watchDir = opts.LogDir
	}
	if err := watcher.Add(watchDir); err != nil {
		logger.Error("failed to watch log directory", field("dir", watchDir), field("error", err))
		os.Exit(1)
	}
	debugf("watching directory: %s", watchDir)

	switchTicker := time.NewTicker(5 * time.Second)
	defer switchTicker.Stop()

	for {
		select {
		case event := <-watcher.Events:
			debugf("fsnotify event: op=%s path=%s", event.Op.String(), event.Name)

			if opts.LogFile == "" && opts.LogDir != "" {
				switched, channel := switchToLatestLog(tailer, opts.LogDir, channels)
				if switched {
					selected.Channel = channel
				}
			}

			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}

			if filepath.Clean(event.Name) != filepath.Clean(tailer.Path) {
				continue
			}

			lines, err := tailer.ReadNewLines()
			if err != nil {
				debugf("failed to read new lines from %s: %v", tailer.Path, err)
				continue
			}
			processLines(client, submitURL, opts.Token, selected.Channel.ID, tailer.Path, lines)
		case err := <-watcher.Errors:
			if err != nil {
				logger.Warn("watcher error", field("error", err))
			}
		case <-switchTicker.C:
			if opts.LogFile == "" && opts.LogDir != "" {
				switched, channel := switchToLatestLog(tailer, opts.LogDir, channels)
				if switched {
					selected.Channel = channel
				}
			}
			// Safety net: some platforms/filesystems may miss fsnotify events.
			lines, err := tailer.ReadNewLines()
			if err != nil {
				debugf("poll read failed for %s: %v", tailer.Path, err)
				continue
			}
			processLines(client, submitURL, opts.Token, selected.Channel.ID, tailer.Path, lines)
		}
	}
}

type Tailer struct {
	Path         string
	Offset       int64
	Encoding     string
	pendingBytes []byte
}

func (t *Tailer) Prime() error {
	file, err := os.Open(t.Path)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	if t.Encoding == "" {
		header := make([]byte, 2)
		n, _ := io.ReadFull(file, header)
		if n == 2 && header[0] == 0xFF && header[1] == 0xFE {
			t.Encoding = "utf16le"
		} else if n == 2 && header[0] == 0xFE && header[1] == 0xFF {
			t.Encoding = "utf16be"
		} else {
			t.Encoding = "utf8"
		}
		debugf("detected log encoding for %s: %s", t.Path, t.Encoding)
	}

	if t.Offset == 0 {
		t.Offset = info.Size()
	}
	return nil
}

func (t *Tailer) ReadNewLines() ([]string, error) {
	file, err := os.Open(t.Path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size() < t.Offset {
		t.Offset = 0
	}

	if _, err := file.Seek(t.Offset, io.SeekStart); err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, err
	}
	t.Offset += int64(buf.Len())

	raw := append([]byte{}, buf.Bytes()...)
	if len(t.pendingBytes) > 0 {
		raw = append(t.pendingBytes, raw...)
		t.pendingBytes = nil
	}
	decoded := decodeLogChunk(raw, t)

	scanner := bufio.NewScanner(strings.NewReader(decoded))
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, nil
}

func decodeLogChunk(raw []byte, t *Tailer) string {
	if len(raw) == 0 {
		return ""
	}

	switch t.Encoding {
	case "utf16le", "utf16be":
		if len(raw)%2 != 0 {
			t.pendingBytes = []byte{raw[len(raw)-1]}
			raw = raw[:len(raw)-1]
		}
		if len(raw) == 0 {
			return ""
		}

		u16 := make([]uint16, 0, len(raw)/2)
		for i := 0; i+1 < len(raw); i += 2 {
			var value uint16
			if t.Encoding == "utf16le" {
				value = binary.LittleEndian.Uint16(raw[i : i+2])
			} else {
				value = binary.BigEndian.Uint16(raw[i : i+2])
			}
			u16 = append(u16, value)
		}
		runes := utf16.Decode(u16)
		if len(runes) > 0 && runes[0] == '\ufeff' {
			runes = runes[1:]
		}
		return string(runes)
	default:
		return string(raw)
	}
}

func postJSON(client *http.Client, url string, payload submitPayload, token string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	debugf("submit payload: %s", clipForLog(body))

	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	debugf("PUT %s -> %s", url, resp.Status)

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("%s: %s", resp.Status, clipForLog(data))
	}
	return nil
}

func fetchChannels(client *http.Client, url string, token string) ([]channelConfig, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	debugf("GET %s -> %s", url, resp.Status)

	data, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("config request failed: %s (content-type: %q, body: %s)", resp.Status, resp.Header.Get("Content-Type"), clipForLog(data))
	}

	var cfg uploaderConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config JSON from %s (content-type: %q, body: %s): %w", url, resp.Header.Get("Content-Type"), clipForLog(data), err)
	}

	out := []channelConfig{}
	for _, channel := range cfg.Channels {
		trimmed := strings.TrimSpace(channel.Name)
		if trimmed != "" {
			out = append(out, channelConfig{
				ID:   strings.TrimSpace(channel.ID),
				Name: trimmed,
			})
		}
	}
	return out, nil
}

func buildAPIBaseURL(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	parsed, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("expected absolute URL like https://example.com")
	}

	path := strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(strings.ToLower(path), "/api") {
		path += "/api"
	}
	parsed.Path = path
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return strings.TrimRight(parsed.String(), "/"), nil
}

func clipForLog(data []byte) string {
	value := strings.TrimSpace(string(data))
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	if value == "" {
		return "<empty>"
	}
	if len(value) > 240 {
		return value[:240] + "..."
	}
	return value
}

func debugf(format string, args ...any) {
	if !debugEnabled || logger == nil {
		return
	}
	logger.Debug(fmt.Sprintf(format, args...))
}

func sendExistingLines(client *http.Client, submitURL string, token string, channelID string, logFile string, cutoff time.Time) error {
	file, err := os.Open(logFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var scanned int
	var submitted int
	for scanner.Scan() {
		scanned++
		line := normalizeLogLine(scanner.Text())
		if !reportPattern.MatchString(line) {
			continue
		}
		when, ok := parseReportTime(line)
		if !ok {
			continue
		}
		if !cutoff.IsZero() && when.Before(cutoff) {
			continue
		}
		payload := submitPayload{
			Text:      line,
			ChannelID: channelID,
		}
		if postErr := postJSON(client, submitURL, payload, token); postErr != nil {
			debugf("failed to submit existing line: %v", postErr)
			continue
		}
		submitted++
	}
	debugf("existing scan complete: scanned=%d submitted=%d file=%s", scanned, submitted, logFile)
	return scanner.Err()
}

func parseReportTime(line string) (time.Time, bool) {
	match := reportPattern.FindStringSubmatch(line)
	if len(match) < 2 {
		return time.Time{}, false
	}
	parsed, err := time.ParseInLocation("2006.01.02 15:04:05", match[1], time.UTC)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func normalizeLogLine(line string) string {
	// EVE logs often contain BOM at line start and CRLF endings.
	line = strings.TrimPrefix(line, "\ufeff")
	return strings.TrimRight(line, "\r")
}

func findLatestLog(dir string, channels []channelConfig) (logSelection, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return logSelection{}, false
	}

	var selection logSelection
	var latestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		channel, ok := resolveChannelForPath(name, channels)
		if !ok {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(latestTime) {
			latestTime = info.ModTime()
			selection = logSelection{
				Path:    filepath.Join(dir, name),
				Channel: channel,
			}
		}
	}
	return selection, selection.Path != ""
}

func switchToLatestLog(tailer *Tailer, logDir string, channels []channelConfig) (bool, channelConfig) {
	latest, ok := findLatestLog(logDir, channels)
	if !ok {
		debugf("no matching log files found in %s", logDir)
		return false, channelConfig{}
	}
	if filepath.Clean(latest.Path) == filepath.Clean(tailer.Path) {
		return false, latest.Channel
	}
	tailer.Path = latest.Path
	tailer.Offset = 0
	if err := tailer.Prime(); err != nil {
		debugf("failed to prime new log file %s: %v", latest.Path, err)
		return false, channelConfig{}
	}
	logger.Info(
		"switched to new log file",
		field("path", latest.Path),
		field("channel", latest.Channel.Name),
		field("channel_id", latest.Channel.ID),
	)
	return true, latest.Channel
}

func processLines(client *http.Client, submitURL string, token string, channelID string, path string, lines []string) {
	if len(lines) == 0 {
		return
	}
	debugf("read %d new lines from %s", len(lines), path)
	for _, line := range lines {
		line = normalizeLogLine(line)
		debugf("line: %s", clipForLog([]byte(line)))

		if !reportPattern.MatchString(line) {
			debugf("skipping non-report line")
			continue
		}

		payload := submitPayload{
			Text:      line,
			ChannelID: channelID,
		}
		if postErr := postJSON(client, submitURL, payload, token); postErr != nil {
			logger.Warn("failed to submit line", field("error", postErr))
			continue
		}
		debugf("report accepted")
	}
}

func resolveChannelForPath(path string, channels []channelConfig) (channelConfig, bool) {
	name := strings.ToLower(filepath.Base(path))
	best := channelConfig{}
	bestLen := 0
	for _, channel := range channels {
		channelName := strings.TrimSpace(channel.Name)
		if channelName == "" {
			continue
		}
		needle := strings.ToLower(channelName)
		if strings.Contains(name, needle) && len(needle) > bestLen {
			best = channel
			bestLen = len(needle)
		}
	}
	return best, bestLen > 0
}
