package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"

	nurl "net/url"

	readeck "codeberg.org/readeck/go-readability/v2"
	htmd "github.com/JohannesKaufmann/html-to-markdown"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

var converter = htmd.NewConverter("", true, nil)

func (s *WebServer) HandleWebFetch(ctx context.Context, params WebFetchParams) (*WebFetchResponse, error) {
	if params.URL == "" {
		return nil, newValidationError("url is required")
	}

	if err := validateFetchURL(params.URL); err != nil {
		return nil, err
	}

	maxLength := params.MaxLength
	if maxLength == 0 {
		maxLength = 5000
	}

	startIndex := params.StartIndex
	if maxLength <= 0 || maxLength >= 1000000 {
		return nil, newValidationError("max_length must be between 1 and 999999")
	}
	if startIndex < 0 {
		return nil, newValidationError("start_index must be >= 0")
	}

	content, prefix, err := fetchURL(ctx, s.httpClient, params.URL, defaultUserAgent, params.Raw, maxLength, startIndex)
	if err != nil {
		slog.Warn("fetch failed", "url", params.URL, "error", err)
		return &WebFetchResponse{URL: params.URL, Error: fmt.Sprintf("Error fetching URL: %v", err)}, nil
	}

	originalLength := len(content)
	var nextStart int
	var truncated bool
	if startIndex >= originalLength {
		content = "<error>No more content available.</error>"
	} else {
		endIndex := min(startIndex+maxLength, originalLength)
		truncatedContent := content[startIndex:endIndex]
		if len(truncatedContent) == 0 {
			content = "<error>No more content available.</error>"
		} else {
			content = truncatedContent
			actualContentLength := len(truncatedContent)
			remainingContent := originalLength - (startIndex + actualContentLength)
			if actualContentLength == maxLength && remainingContent > 0 {
				nextStart = startIndex + actualContentLength
				truncated = true
			}
		}
	}

	contentType := "raw"
	if prefix == "Markdown" {
		contentType = "markdown"
	}

	response := &WebFetchResponse{
		URL:            params.URL,
		Content:        content,
		ContentType:    contentType,
		OriginalLength: originalLength,
		Truncated:      truncated,
		NextStart:      nextStart,
	}
	if nextStart > 0 {
		response.NextStart = nextStart
	}

	return response, nil
}

// extractContentFromHTML extracts text from HTML and converts to Markdown
func extractContentFromHTML(htmlContent, uri string) string {
	parsedURL, _ := nurl.Parse(uri)
	article, err := readeck.FromReader(strings.NewReader(htmlContent), parsedURL)
	if err != nil {
		return "<error>Page failed to be simplified from HTML</error>"
	}

	if article.Node == nil {
		return "<error>Page failed to be simplified from HTML</error>"
	}

	var sb strings.Builder
	if err := article.RenderText(&sb); err != nil {
		return "<error>Failed to render article text</error>"
	}

	textContent := sb.String()
	if textContent == "" {
		return "<error>Page failed to be simplified from HTML</error>"
	}

	markdown, err := converter.ConvertString(textContent)
	if err != nil {
		slog.Info("failed to convert HTML to markdown", "err", err)
		return "<error>Failed to convert HTML to markdown</error>"
	}

	return markdown
}

// validateFetchURL validates that a URL is safe to fetch (prevents SSRF).
// Overridable in tests.
var validateFetchURL = func(rawURL string) error {
	u, err := nurl.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return newValidationError("only http and https schemes are allowed")
	}
	if u.Host == "" {
		return newValidationError("URL must include a host")
	}
	host, _, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("failed to resolve host: %w", err)
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() {
			return newValidationError("internal/private IP addresses are not allowed")
		}
	}
	return nil
}

// fetchURL fetches web page content, supports HTML to Markdown conversion.
func fetchURL(ctx context.Context, client *http.Client, urlStr, userAgent string, raw bool, maxLength, startIndex int) (content, prefix string, err error) {
	slog.Debug("fetching URL", "url", urlStr, "raw", raw)

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		err = fmt.Errorf("HTTP %d", resp.StatusCode)
		return
	}

	readLimit := int64(startIndex + maxLength + 1) // +1 to detect truncation
	b, err := io.ReadAll(io.LimitReader(resp.Body, readLimit))
	if err != nil {
		return
	}
	content = string(b)
	slog.Debug("fetch completed", "url", urlStr, "status", resp.StatusCode, "content_length", len(content))

	contentType := resp.Header.Get("content-type")
	pagePreview := content
	if len(pagePreview) > 100 {
		pagePreview = pagePreview[:100]
	}
	isHTML := strings.Contains(contentType, "text/html") || strings.Contains(pagePreview, "<html")

	if isHTML && !raw {
		content = extractContentFromHTML(content, urlStr)
		prefix = "Markdown"
	} else {
		prefix = fmt.Sprintf("Content type %s, raw content:", contentType)
	}
	return
}
