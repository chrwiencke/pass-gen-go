package shortcut

import (
	"fmt"
	"runtime"
	"strings"
	"unicode"
)

type Shortcut struct {
	Control bool
	Command bool
	Windows bool
	Key     string
}

func Default() string {
	if runtime.GOOS == "darwin" {
		return "Ctrl+Command+P"
	}
	return "Ctrl+Windows+P"
}

func Normalize(value string) string {
	parsed, err := ParseCurrentPlatform(value)
	if err != nil {
		return Default()
	}
	return parsed.String()
}

func ParseCurrentPlatform(value string) (Shortcut, error) {
	parsed, err := Parse(value)
	if err != nil {
		return Shortcut{}, err
	}
	switch runtime.GOOS {
	case "darwin":
		if parsed.Windows {
			return Shortcut{}, fmt.Errorf("the Windows modifier is not available on macOS")
		}
	case "windows":
		if parsed.Command {
			return Shortcut{}, fmt.Errorf("the Command modifier is not available on Windows")
		}
	}
	return parsed, nil
}

func Parse(value string) (Shortcut, error) {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '+' || unicode.IsSpace(r)
	})

	var parsed Shortcut
	for _, part := range parts {
		token := strings.ToLower(strings.TrimSpace(part))
		if token == "" {
			continue
		}

		switch token {
		case "ctrl", "control":
			parsed.Control = true
		case "cmd", "command", "meta":
			parsed.Command = true
		case "win", "windows", "super":
			parsed.Windows = true
		default:
			if parsed.Key != "" {
				return Shortcut{}, fmt.Errorf("shortcut must contain one key")
			}
			key := strings.ToUpper(token)
			if len(key) != 1 || !isSupportedKey(rune(key[0])) {
				return Shortcut{}, fmt.Errorf("shortcut key must be A-Z or 0-9")
			}
			parsed.Key = key
		}
	}

	if parsed.Key == "" {
		return Shortcut{}, fmt.Errorf("shortcut must contain a key")
	}
	if !parsed.Control && !parsed.Command && !parsed.Windows {
		return Shortcut{}, fmt.Errorf("shortcut must contain at least one modifier")
	}
	return parsed, nil
}

func (s Shortcut) String() string {
	parts := make([]string, 0, 4)
	if s.Control {
		parts = append(parts, "Ctrl")
	}
	if s.Command {
		parts = append(parts, "Command")
	}
	if s.Windows {
		parts = append(parts, "Windows")
	}
	parts = append(parts, s.Key)
	return strings.Join(parts, "+")
}

func isSupportedKey(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
