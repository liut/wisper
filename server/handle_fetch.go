package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	nurl "net/url"

	htmd "github.com/JohannesKaufmann/html-to-markdown"
	readeck "codeberg.org/readeck/go-readability/v2"
)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

var converter = htmd.NewConverter("", true, nil)

// handleWebFetch handles the web_fetch tool
func (s *WebServer) handleWebFetch(ctx context.Context, params WebFetchParams) (*webFetchOutput, error) {
	if params.URL == "" {
		return nil, errors.New("url is required")
	}

	maxLength := params.MaxLength
	if maxLength == 0 {
		maxLength = 5000
	}

	startIndex := params.StartIndex
	if maxLength <= 0 || maxLength >= 1000000 {
		return nil, errors.New("max_length must be between 1 and 999999")
	}
	if startIndex < 0 {
		return nil, errors.New("start_index must be >= 0")
	}

	content, prefix, err := fetchURL(ctx, params.URL, defaultUserAgent, params.Raw)
	if err != nil {
		slog.Warn("fetch failed", "url", params.URL, "error", err)
		return &webFetchOutput{Text: fmt.Sprintf("Error fetching URL: %v", err)}, nil
	}

	// Handle truncation
	originalLength := len(content)
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
				nextStart := startIndex + actualContentLength
				content += fmt.Sprintf("\n\n<error>Content truncated. Call the fetch tool with a start_index of %d to get more content.</error>", nextStart)
			}
		}
	}

	resultText := fmt.Sprintf("%s\nContents of %s:\n%s", prefix, params.URL, content)

	return &webFetchOutput{Text: resultText}, nil
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

// fetchURL fetches web page content, supports HTML to Markdown conversion
func fetchURL(ctx context.Context, urlStr, userAgent string, raw bool) (content, prefix string, err error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

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

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	content = string(b)

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
