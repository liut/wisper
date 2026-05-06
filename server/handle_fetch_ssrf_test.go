package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	nurl "net/url"
)

// restoreValidation restores the default SSRF validation function.
func restoreValidation() {
	validateFetchURL = func(rawURL string) error {
		u, err := nurl.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return errors.New("only http and https schemes are allowed")
		}
		if u.Host == "" {
			return errors.New("URL must include a host")
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
				return errors.New("internal/private IP addresses are not allowed")
			}
		}
		return nil
	}
}

func TestValidateFetchURL_RejectsFileScheme(t *testing.T) {
	restoreValidation()
	err := validateFetchURL("file:///etc/passwd")
	if err == nil {
		t.Fatal("expected error for file:// scheme, got nil")
	}
}

func TestValidateFetchURL_RejectsLoopback(t *testing.T) {
	restoreValidation()
	err := validateFetchURL("http://127.0.0.1:8080/")
	if err == nil {
		t.Fatal("expected error for loopback IP, got nil")
	}
}

func TestValidateFetchURL_RejectsPrivateIP(t *testing.T) {
	restoreValidation()
	err := validateFetchURL("http://10.0.0.1/")
	if err == nil {
		t.Fatal("expected error for private IP, got nil")
	}
}

func TestValidateFetchURL_RejectsLinkLocal(t *testing.T) {
	restoreValidation()
	err := validateFetchURL("http://169.254.169.254/latest/meta-data/")
	if err == nil {
		t.Fatal("expected error for link-local IP, got nil")
	}
}

func TestValidateFetchURL_RejectsInvalidURL(t *testing.T) {
	restoreValidation()
	err := validateFetchURL("not-a-valid-url%%%")
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}

func TestValidateFetchURL_AllowsHTTPS(t *testing.T) {
	restoreValidation()
	err := validateFetchURL("https://example.com/")
	if err != nil {
		t.Fatalf("expected no error for valid https URL, got: %v", err)
	}
}

func TestHandleWebFetch_RejectsFileURL(t *testing.T) {
	restoreValidation()
	s := testWebServer()
	_, err := s.HandleWebFetch(context.Background(), WebFetchParams{URL: "file:///etc/passwd"})
	if err == nil {
		t.Fatal("expected error for file:// URL, got nil")
	}
}
