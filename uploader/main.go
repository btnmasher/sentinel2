package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	flags "github.com/jessevdk/go-flags"
	"github.com/joho/godotenv"
)

type Options struct {
	BaseURL string `long:"base-url" env:"SENTINEL_BASE_URL" description:"Sentinel base URL (e.g. https://intel.example.com)" required:"true"`
	Token   string `long:"token" env:"SENTINEL_TOKEN" description:"Uploader token" required:"true"`
	LogFile string `long:"log-file" env:"SENTINEL_LOG_FILE" description:"EVE chat log file to watch"`
	LogDir  string `long:"log-dir" env:"SENTINEL_LOG_DIR" description:"Directory containing EVE chat logs"`
}

var reportPattern = regexp.MustCompile(`\[ (?P<date>.*) \] (?P<author>[\s\w\-']+) > (?P<text>.*)`)

type submitPayload struct {
	Text   string `json:"text"`
	Status string `json:"status"`
}

type uploaderConfig struct {
	Channels []string `json:"channels"`
}

func main() {
	_ = godotenv.Load()

	opts := Options{}
	if _, err := flags.Parse(&opts); err != nil {
		os.Exit(1)
	}

	baseURL := strings.TrimRight(opts.BaseURL, "/")
	configURL := baseURL + "/uploader/config"
	submitURL := baseURL + "/uploader/submit"

	if opts.LogDir == "" && opts.LogFile == "" {
		opts.LogDir = defaultLogDir()
		if opts.LogDir == "" {
			fmt.Println("no default log directory found; set --log-dir or --log-file")
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	channels, err := fetchChannels(client, configURL, opts.Token)
	if err != nil {
		fmt.Println("failed to fetch channels:", err)
		os.Exit(1)
	}
	if len(channels) == 0 {
		fmt.Println("no channels configured")
		os.Exit(1)
	}

	logFile := opts.LogFile
	if logFile == "" && opts.LogDir != "" {
		logFile = findLatestLog(opts.LogDir, channels)
	}
	if logFile == "" {
		fmt.Println("missing log file")
		os.Exit(1)
	}

	cutoff := time.Now().Add(-1 * time.Minute)
	if err := sendExistingLines(client, submitURL, opts.Token, logFile, channels, cutoff); err != nil {
		fmt.Println("failed to read recent logs:", err)
	}

	tailer := &Tailer{Path: logFile}
	_ = tailer.Prime()

	fmt.Printf("Watching %s\n", logFile)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Println("failed to watch logs")
		os.Exit(1)
	}
	defer watcher.Close()

	watchDir := filepath.Dir(logFile)
	if err := watcher.Add(watchDir); err != nil {
		fmt.Println("failed to watch log directory")
		os.Exit(1)
	}

	for {
		select {
		case event := <-watcher.Events:
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) == 0 {
				continue
			}

			if opts.LogFile == "" && opts.LogDir != "" {
				if latest := findLatestLog(opts.LogDir, channels); latest != "" && latest != tailer.Path {
					tailer.Path = latest
					tailer.Offset = 0
					_ = tailer.Prime()
					fmt.Printf("Switched to %s\n", latest)
				}
			}

			if filepath.Clean(event.Name) != filepath.Clean(tailer.Path) {
				continue
			}

			lines, err := tailer.ReadNewLines()
			if err != nil {
				continue
			}

			for _, line := range lines {
				if !shouldSend(line, channels) {
					continue
				}

				if !reportPattern.MatchString(line) {
					continue
				}

				payload := submitPayload{
					Text:   line,
					Status: "Running",
				}
				_ = postJSON(client, submitURL, payload, opts.Token)
			}
		case err := <-watcher.Errors:
			if err != nil {
				fmt.Println("watcher error:", err)
			}
		}
	}
}

func shouldSend(line string, channels []string) bool {
	lower := strings.ToLower(line)
	for _, channel := range channels {
		if strings.Contains(lower, strings.ToLower(channel)) {
			return true
		}
	}
	return false
}

type Tailer struct {
	Path   string
	Offset int64
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

	scanner := bufio.NewScanner(buf)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, nil
}

func postJSON(client *http.Client, url string, payload submitPayload, token string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

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

	if resp.StatusCode >= 400 {
		return errors.New(resp.Status)
	}
	return nil
}

func fetchChannels(client *http.Client, url string, token string) ([]string, error) {
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

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("config request failed: %s", resp.Status)
	}

	var cfg uploaderConfig
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, err
	}

	out := []string{}
	for _, channel := range cfg.Channels {
		trimmed := strings.TrimSpace(channel)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out, nil
}

func sendExistingLines(client *http.Client, submitURL string, token string, logFile string, channels []string, cutoff time.Time) error {
	file, err := os.Open(logFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
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
		if !shouldSend(line, channels) {
			continue
		}
		payload := submitPayload{
			Text:   line,
			Status: "Running",
		}
		_ = postJSON(client, submitURL, payload, token)
	}
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

func findLatestLog(dir string, channels []string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var candidate string
	var latestTime time.Time
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lowerName := strings.ToLower(name)
		for _, channel := range channels {
			if strings.Contains(lowerName, strings.ToLower(channel)) {
				info, err := entry.Info()
				if err != nil {
					continue
				}
				if info.ModTime().After(latestTime) {
					latestTime = info.ModTime()
					candidate = filepath.Join(dir, name)
				}
			}
		}
	}
	return candidate
}
