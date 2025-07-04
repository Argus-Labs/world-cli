package validate

import (
	"strings"
	"unicode"

	"github.com/rotisserie/eris"
	"pkg.world.dev/world-cli/common/printer"
)

func Name(name string, maxLength int) error {
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
