package utils

import (
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/printer"
)

var ErrNotInWorldCardinalRoot = eris.New("Not in a World Cardinal root")

func ValidateName(name string, maxLength int) error {
	if name == "" {
		printer.Errorln("Name cannot be empty")
		printer.NewLine(1)
		return eris.New("empty name")
	}

	if len(name) > maxLength {
		printer.Errorf("Name cannot be longer than %d characters\n", maxLength)
		printer.NewLine(1)
		return eris.New("name too long")
	}

	if strings.ContainsAny(name, "<>:\"/\\|?*") {
		printer.Errorln("Name contains invalid characters" +
			"   Invalid characters: < > : \" / \\ | ? *")
		printer.NewLine(1)
		return eris.New("invalid characters")
	}

	for i, r := range name {
		if !unicode.IsPrint(r) {
			printer.Errorf("Name contains non-printable characters at index %d\n", i)
			printer.NewLine(1)
			return eris.New("non-printable character in name")
		}
	}
	return nil
}

func IsValidURL(urlStr string) error {
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return eris.New("Invalid URL: Must start with http:// or https://")
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

// IsInWorldCardinalRoot checks if the current working directory is a World project.
// It checks for the presence of world.toml and cardinal directory.
func IsInWorldCardinalRoot() (bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return false, eris.Wrap(err, "failed to get working directory")
	}

	worldTomlPath := filepath.Join(cwd, "world.toml")
	cardinalDirPath := filepath.Join(cwd, "cardinal")

	tomlInfo, err := os.Stat(worldTomlPath)
	if err != nil || tomlInfo.IsDir() {
		return false, nil //nolint:nilerr // false return gives all the information we need
	}

	cardinalInfo, err := os.Stat(cardinalDirPath)
	if err != nil || !cardinalInfo.IsDir() {
		return false, nil //nolint:nilerr // false return gives all the information we need
	}
	return true, nil
}
