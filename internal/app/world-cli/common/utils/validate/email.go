package validate

import (
	"net/mail"
	"strings"

	"github.com/rotisserie/eris"
	"golang.org/x/net/idna"
	"pkg.world.dev/world-cli/internal/pkg/printer"
)

// Email checks if an email address is syntactically valid
// and supports internationalized domain names (RFC 6531).
func Email(email string) error {
	if email == "" {
		printer.Errorln("Email cannot be empty")
		printer.NewLine(1)
		return eris.New("email cannot be empty")
	}

	// Parse the email address for syntax validation
	addr, err := mail.ParseAddress(email)
	if err != nil {
		printer.Errorln("Invalid email format")
		printer.Infoln("Email must be in the format: user@domain.com")
		printer.NewLine(1)
		return eris.Wrap(err, "invalid email syntax")
	}

	// Split address into local part and domain
	parts := strings.Split(addr.Address, "@")
	localPart := parts[0]
	domainPart := parts[1]

	// Validate local part length
	if len(localPart) > 64 {
		printer.Errorln("Invalid email format: local part too long (max 64 characters)")
		printer.NewLine(1)
		return eris.New("local part too long: maximum 64 characters")
	}

	// Normalize domain to ASCII (Punycode) to support Unicode domains
	normalizedDomain, err := idna.Lookup.ToASCII(domainPart)
	if err != nil {
		printer.Errorf("Invalid email format: domain '%s' contains invalid characters\n", domainPart)
		printer.NewLine(1)
		return eris.Wrap(err, "invalid domain (Unicode normalization failed)")
	}

	// Validate domain part length
	if len(normalizedDomain) > 253 {
		printer.Errorln("Invalid email format: domain too long (max 253 characters)")
		printer.NewLine(1)
		return eris.New("domain too long: maximum 253 characters")
	}

	return nil
}
