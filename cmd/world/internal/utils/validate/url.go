package validate

import (
	"net"
	"net/url"
	"strings"

	"github.com/rotisserie/eris"
)

func IsURL(urlStr string) error {
	// Check for empty URL
	if urlStr == "" {
		return eris.New("Invalid URL: URL cannot be empty")
	}

	// Check for spaces in the URL
	if strings.Contains(urlStr, " ") {
		return eris.New("Invalid URL: Cannot contain spaces")
	}

	// Check for invalid characters
	if strings.ContainsAny(urlStr, "<>\"{}|\\^`") {
		return eris.New("Invalid URL: Contains invalid characters")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return eris.Wrap(err, "Invalid URL")
	}

	// Check if protocol is http or https
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return eris.New("Invalid URL: Must start with http:// or https://")
	}

	// Check if hostname is empty
	if parsedURL.Hostname() == "" {
		return eris.New("Invalid URL: Must have a hostname")
	}

	// Check if hostname has a TLD (at least one dot with characters after it)
	hostname := parsedURL.Hostname()
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 || parts[len(parts)-1] == "" {
		return eris.New("Invalid URL: Must have a TLD")
	}

	// Skip DNS lookup for localhost
	if hostname == "localhost" {
		return eris.New("Invalid URL: Cannot use localhost")
	}

	// Perform DNS lookup to verify domain exists
	_, err = net.LookupHost(hostname)
	return eris.Wrap(err, "Invalid URL")
}
