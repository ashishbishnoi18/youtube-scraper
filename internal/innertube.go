package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	WebAPIKey = "AIzaSyAO_FJ2SlqU8Q4STEHLGCilw_Y9_11qcW8"
	BaseURL   = "https://www.youtube.com/youtubei/v1"

	WebClientVersion = "2.20260301.00.00"
)

// InnerTubeClient handles communication with YouTube's InnerTube API.
type InnerTubeClient struct {
	HTTP *http.Client
}

// NewInnerTubeClient creates a new InnerTube client.
func NewInnerTubeClient(httpClient *http.Client) *InnerTubeClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &InnerTubeClient{HTTP: httpClient}
}

func (c *InnerTubeClient) webContext() map[string]interface{} {
	return map[string]interface{}{
		"client": map[string]interface{}{
			"clientName":    "WEB",
			"clientVersion": WebClientVersion,
			"hl":            "en",
			"gl":            "US",
		},
	}
}

func (c *InnerTubeClient) iosContext() map[string]interface{} {
	return map[string]interface{}{
		"client": map[string]interface{}{
			"clientName":       "IOS",
			"clientVersion":    "19.29.1",
			"deviceMake":       "Apple",
			"deviceModel":      "iPhone16,2",
			"hl":               "en",
			"gl":               "US",
			"osName":           "iPhone",
			"osVersion":        "17.5.1.21F90",
			"userAgent":        "com.google.ios.youtube/19.29.1 (iPhone16,2; U; CPU iOS 17_5_1 like Mac OS X;)",
		},
	}
}

// Request makes a request to the InnerTube API using the WEB client.
func (c *InnerTubeClient) Request(ctx context.Context, endpoint string, payload map[string]interface{}) (map[string]interface{}, error) {
	return c.RequestWithContext(ctx, endpoint, payload, c.webContext())
}

// RequestIOS makes a request using the IOS client (avoids LOGIN_REQUIRED on player).
func (c *InnerTubeClient) RequestIOS(ctx context.Context, endpoint string, payload map[string]interface{}) (map[string]interface{}, error) {
	return c.RequestWithContext(ctx, endpoint, payload, c.iosContext())
}

// RequestWithContext makes a request to the InnerTube API with a specific client context.
func (c *InnerTubeClient) RequestWithContext(ctx context.Context, endpoint string, payload map[string]interface{}, clientCtx map[string]interface{}) (map[string]interface{}, error) {
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["context"] = clientCtx

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	reqURL := fmt.Sprintf("%s/%s?key=%s", BaseURL, endpoint, WebAPIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "https://www.youtube.com")
	req.Header.Set("Referer", "https://www.youtube.com/")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

func (c *InnerTubeClient) Player(ctx context.Context, videoID string) (map[string]interface{}, error) {
	return c.Request(ctx, "player", map[string]interface{}{"videoId": videoID})
}

func (c *InnerTubeClient) Next(ctx context.Context, videoID string) (map[string]interface{}, error) {
	return c.Request(ctx, "next", map[string]interface{}{"videoId": videoID})
}

func (c *InnerTubeClient) NextWithContinuation(ctx context.Context, continuation string) (map[string]interface{}, error) {
	return c.Request(ctx, "next", map[string]interface{}{"continuation": continuation})
}

func (c *InnerTubeClient) Browse(ctx context.Context, browseID string, params string) (map[string]interface{}, error) {
	payload := map[string]interface{}{"browseId": browseID}
	if params != "" {
		payload["params"] = params
	}
	return c.Request(ctx, "browse", payload)
}

func (c *InnerTubeClient) BrowseWithContinuation(ctx context.Context, continuation string) (map[string]interface{}, error) {
	return c.Request(ctx, "browse", map[string]interface{}{"continuation": continuation})
}

func (c *InnerTubeClient) Search(ctx context.Context, query string, params string) (map[string]interface{}, error) {
	payload := map[string]interface{}{"query": query}
	if params != "" {
		payload["params"] = params
	}
	return c.Request(ctx, "search", payload)
}

func (c *InnerTubeClient) SearchWithContinuation(ctx context.Context, continuation string) (map[string]interface{}, error) {
	return c.Request(ctx, "search", map[string]interface{}{"continuation": continuation})
}

func (c *InnerTubeClient) ResolveURL(ctx context.Context, ytURL string) (map[string]interface{}, error) {
	return c.Request(ctx, "navigation/resolve_url", map[string]interface{}{"url": ytURL})
}

// --- Helpers ---

func GetString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func GetInt(m map[string]interface{}, key string) int64 {
	switch v := m[key].(type) {
	case string:
		var i int64
		fmt.Sscanf(v, "%d", &i)
		return i
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	}
	return 0
}

func GetFloat(m map[string]interface{}, key string) float64 {
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

func GetBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func GetTextFromRuns(m map[string]interface{}, key string) string {
	if textObj, ok := m[key].(map[string]interface{}); ok {
		if runs, ok := textObj["runs"].([]interface{}); ok {
			var texts []string
			for _, run := range runs {
				if runMap, ok := run.(map[string]interface{}); ok {
					if text, ok := runMap["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
			return strings.Join(texts, "")
		}
		if simpleText, ok := textObj["simpleText"].(string); ok {
			return simpleText
		}
	}
	return ""
}

func GetTextFromSimpleText(m map[string]interface{}, key string) string {
	if textObj, ok := m[key].(map[string]interface{}); ok {
		if simpleText, ok := textObj["simpleText"].(string); ok {
			return simpleText
		}
		if runs, ok := textObj["runs"].([]interface{}); ok {
			var texts []string
			for _, run := range runs {
				if runMap, ok := run.(map[string]interface{}); ok {
					if text, ok := runMap["text"].(string); ok {
						texts = append(texts, text)
					}
				}
			}
			return strings.Join(texts, "")
		}
	}
	return ""
}

func GetThumbnails(renderer map[string]interface{}) []interface{} {
	if thumbnail, ok := renderer["thumbnail"].(map[string]interface{}); ok {
		if thumbs, ok := thumbnail["thumbnails"].([]interface{}); ok {
			return thumbs
		}
	}
	return nil
}

func ExtractContinuationToken(contItem map[string]interface{}) string {
	if contEndpoint, ok := contItem["continuationEndpoint"].(map[string]interface{}); ok {
		if contCommand, ok := contEndpoint["continuationCommand"].(map[string]interface{}); ok {
			return GetString(contCommand, "token")
		}
	}
	return ""
}

func ExtractContinuationFromRenderer(contRenderer map[string]interface{}) string {
	if contEndpoint, ok := contRenderer["continuationEndpoint"].(map[string]interface{}); ok {
		if contCommand, ok := contEndpoint["continuationCommand"].(map[string]interface{}); ok {
			return GetString(contCommand, "token")
		}
	}
	if button, ok := contRenderer["button"].(map[string]interface{}); ok {
		if buttonRenderer, ok := button["buttonRenderer"].(map[string]interface{}); ok {
			if command, ok := buttonRenderer["command"].(map[string]interface{}); ok {
				if contCommand, ok := command["continuationCommand"].(map[string]interface{}); ok {
					return GetString(contCommand, "token")
				}
			}
		}
	}
	return ""
}

// --- URL extraction ---

var (
	VideoIDRegexes = []*regexp.Regexp{
		regexp.MustCompile(`(?:youtube\.com\/watch\?v=|youtu\.be\/|youtube\.com\/embed\/|youtube\.com\/v\/|youtube\.com\/shorts\/)([a-zA-Z0-9_-]{11})`),
		regexp.MustCompile(`^([a-zA-Z0-9_-]{11})$`),
	}
	ChannelIDRegex  = regexp.MustCompile(`^UC[a-zA-Z0-9_-]{22}$`)
	HandleRegex     = regexp.MustCompile(`(?:youtube\.com\/@|^@)([a-zA-Z0-9_.-]+)`)
	ChannelURLRegex = regexp.MustCompile(`youtube\.com\/channel\/(UC[a-zA-Z0-9_-]{22})`)
	CustomURLRegex  = regexp.MustCompile(`youtube\.com\/c\/([^\/\?]+)`)
	UserURLRegex    = regexp.MustCompile(`youtube\.com\/user\/([^\/\?]+)`)
	PlaylistIDRegex = regexp.MustCompile(`(?:list=|playlist\?list=)([a-zA-Z0-9_-]+)`)
	DirectPlaylistRegex = regexp.MustCompile(`^(PL[a-zA-Z0-9_-]+|UU[a-zA-Z0-9_-]+|OL[a-zA-Z0-9_-]+|LL|WL|FL[a-zA-Z0-9_-]+|RD[a-zA-Z0-9_-]*)$`)
)

func ExtractVideoID(input string) string {
	input = strings.TrimSpace(input)
	for _, re := range VideoIDRegexes {
		if matches := re.FindStringSubmatch(input); len(matches) > 1 {
			return matches[1]
		}
	}
	if u, err := url.Parse(input); err == nil {
		if v := u.Query().Get("v"); v != "" && len(v) == 11 {
			return v
		}
	}
	return ""
}

func ExtractPlaylistID(input string) string {
	input = strings.TrimSpace(input)
	if DirectPlaylistRegex.MatchString(input) {
		return input
	}
	if matches := PlaylistIDRegex.FindStringSubmatch(input); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// ResolveChannelID resolves various channel URL formats to a channel ID.
func ResolveChannelID(ctx context.Context, client *InnerTubeClient, input string) (string, error) {
	input = strings.TrimSpace(input)

	if ChannelIDRegex.MatchString(input) {
		return input, nil
	}
	if matches := ChannelURLRegex.FindStringSubmatch(input); len(matches) > 1 {
		return matches[1], nil
	}
	if matches := HandleRegex.FindStringSubmatch(input); len(matches) > 1 {
		return resolveHandle(ctx, client, matches[1])
	}
	if matches := CustomURLRegex.FindStringSubmatch(input); len(matches) > 1 {
		return resolveViaURL(ctx, client, "https://www.youtube.com/c/"+matches[1])
	}
	if matches := UserURLRegex.FindStringSubmatch(input); len(matches) > 1 {
		return resolveViaURL(ctx, client, "https://www.youtube.com/user/"+matches[1])
	}
	if u, err := url.Parse(input); err == nil && u.Host != "" {
		return resolveViaURL(ctx, client, input)
	}
	return resolveHandle(ctx, client, input)
}

func resolveHandle(ctx context.Context, client *InnerTubeClient, handle string) (string, error) {
	handle = strings.TrimPrefix(handle, "@")
	result, err := client.ResolveURL(ctx, "https://www.youtube.com/@"+handle)
	if err != nil {
		return "", err
	}
	return extractBrowseIDFromResolve(result)
}

func resolveViaURL(ctx context.Context, client *InnerTubeClient, ytURL string) (string, error) {
	result, err := client.ResolveURL(ctx, ytURL)
	if err != nil {
		return "", err
	}
	return extractBrowseIDFromResolve(result)
}

func extractBrowseIDFromResolve(result map[string]interface{}) (string, error) {
	if endpoint, ok := result["endpoint"].(map[string]interface{}); ok {
		if browseEndpoint, ok := endpoint["browseEndpoint"].(map[string]interface{}); ok {
			if browseID, ok := browseEndpoint["browseId"].(string); ok {
				return browseID, nil
			}
		}
	}
	return "", fmt.Errorf("channel not found")
}

func ExtractChannelIDFromOwner(renderer map[string]interface{}) string {
	if ownerText, ok := renderer["ownerText"].(map[string]interface{}); ok {
		if runs, ok := ownerText["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				if navEndpoint, ok := run["navigationEndpoint"].(map[string]interface{}); ok {
					if browseEndpoint, ok := navEndpoint["browseEndpoint"].(map[string]interface{}); ok {
						return GetString(browseEndpoint, "browseId")
					}
				}
			}
		}
	}
	return ""
}

func ExtractChannelIDFromShortByline(renderer map[string]interface{}) string {
	if shortByline, ok := renderer["shortBylineText"].(map[string]interface{}); ok {
		if runs, ok := shortByline["runs"].([]interface{}); ok && len(runs) > 0 {
			if run, ok := runs[0].(map[string]interface{}); ok {
				if navEndpoint, ok := run["navigationEndpoint"].(map[string]interface{}); ok {
					if browseEndpoint, ok := navEndpoint["browseEndpoint"].(map[string]interface{}); ok {
						return GetString(browseEndpoint, "browseId")
					}
				}
			}
		}
	}
	return ""
}

func CheckIsLive(renderer map[string]interface{}) bool {
	if badges, ok := renderer["badges"].([]interface{}); ok {
		for _, badge := range badges {
			if badgeMap, ok := badge.(map[string]interface{}); ok {
				if metadataBadge, ok := badgeMap["metadataBadgeRenderer"].(map[string]interface{}); ok {
					if GetString(metadataBadge, "style") == "BADGE_STYLE_TYPE_LIVE_NOW" {
						return true
					}
				}
			}
		}
	}
	return false
}

func DecodeHTMLEntities(s string) string {
	replacer := strings.NewReplacer(
		"&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"",
		"&#39;", "'", "&apos;", "'", "&#x27;", "'", "&#x2F;", "/", "&nbsp;", " ",
	)
	return replacer.Replace(s)
}

// DoJSON performs an HTTP request and decodes JSON response.
func DoJSON(client *http.Client, req *http.Request, dst interface{}) error {
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("not found")
	}
	if resp.StatusCode == 429 {
		return fmt.Errorf("rate limited")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, body)
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
