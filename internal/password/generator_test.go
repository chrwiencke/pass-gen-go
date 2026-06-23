package password

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerateShapeLengthCapitalizationAndASCII(t *testing.T) {
	pattern := regexp.MustCompile(`^[A-Z][a-z]+-[A-Z][a-z]+[0-9]$`)

	for i := 0; i < 1000; i++ {
		pw, err := Generate()
		if err != nil {
			t.Fatalf("Generate() returned error: %v", err)
		}
		if len(pw) < MinLength || len(pw) > MaxLength {
			t.Fatalf("password %q length is %d, want between %d and %d", pw, len(pw), MinLength, MaxLength)
		}
		if !pattern.MatchString(pw) {
			t.Fatalf("password %q does not match Word-WordDigit", pw)
		}
		if !isPlainASCII(pw) {
			t.Fatalf("password %q is not plain ASCII", pw)
		}
	}
}

func TestWordListIsLowercasePlainASCIIAndLarge(t *testing.T) {
	if len(norwegianWords) < 1000 {
		t.Fatalf("word list has %d words, want at least 1000", len(norwegianWords))
	}

	seen := map[string]bool{}
	for _, word := range norwegianWords {
		if word == "" {
			t.Fatal("word list contains an empty word")
		}
		if !isPlainASCII(word) {
			t.Fatalf("word %q is not plain ASCII", word)
		}
		if word != strings.ToLower(word) {
			t.Fatalf("word %q is not lowercase", word)
		}
		if seen[word] {
			t.Fatalf("duplicate word %q", word)
		}
		seen[word] = true
	}
}

func TestCapitalizeWord(t *testing.T) {
	cases := map[string]string{
		"fjell":   "Fjell",
		"OVENFOR": "Ovenfor",
		"bIL":     "Bil",
	}

	for input, want := range cases {
		if got := capitalizeWord(input); got != want {
			t.Fatalf("capitalizeWord(%q) = %q, want %q", input, got, want)
		}
	}
}
