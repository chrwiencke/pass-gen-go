package password

import (
	"fmt"
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
	assertWordList(t, "norwegian", norwegianWords, 1000)
	assertWordList(t, "english", englishWords, 250)
}

func TestGeneratePassphraseHonorsCustomShortLength(t *testing.T) {
	settings := Settings{
		Mode:      ModePassphrase,
		Language:  LanguageNorwegian,
		MinLength: 6,
		MaxLength: 8,
		Lowercase: true,
		Uppercase: false,
		Numbers:   false,
		Special:   false,
	}

	pw, err := generateWithSettingsWithRand(settings, fixedInts(
		0,
		mustWordIndex(t, norwegianWords, "and"),
		mustWordIndex(t, norwegianWords, "arm"),
	))
	if err != nil {
		t.Fatalf("generateWithSettingsWithRand() returned error: %v", err)
	}
	if pw != "andarm" {
		t.Fatalf("password = %q, want %q", pw, "andarm")
	}
}

func TestGenerateEnglishPassphrase(t *testing.T) {
	settings := Settings{
		Mode:      ModePassphrase,
		Language:  LanguageEnglish,
		MinLength: 10,
		MaxLength: 20,
		Lowercase: true,
		Uppercase: true,
		Numbers:   false,
		Special:   true,
	}

	pw, err := generateWithSettingsWithRand(settings, fixedInts(
		0,
		mustWordIndex(t, englishWords, "apple"),
		mustWordIndex(t, englishWords, "bridge"),
	))
	if err != nil {
		t.Fatalf("generateWithSettingsWithRand() returned error: %v", err)
	}
	if pw != "Apple-Bridge" {
		t.Fatalf("password = %q, want %q", pw, "Apple-Bridge")
	}
}

func TestGenerateRandomPasswordUsesSelectedGroups(t *testing.T) {
	settings := Settings{
		Mode:      ModeRandom,
		Language:  LanguageNorwegian,
		MinLength: 12,
		MaxLength: 12,
		Lowercase: true,
		Uppercase: true,
		Numbers:   true,
		Special:   true,
	}

	pw, err := GenerateWithSettings(settings)
	if err != nil {
		t.Fatalf("GenerateWithSettings() returned error: %v", err)
	}
	if len(pw) != 12 {
		t.Fatalf("password %q length is %d, want 12", pw, len(pw))
	}
	assertMatches(t, pw, `[a-z]`)
	assertMatches(t, pw, `[A-Z]`)
	assertMatches(t, pw, `[0-9]`)
	assertMatches(t, pw, `[!@#$%^&*_\-+=?]`)
}

func TestValidateRejectsRandomPasswordWithoutCharacters(t *testing.T) {
	settings := DefaultSettings()
	settings.Mode = ModeRandom
	settings.Lowercase = false
	settings.Uppercase = false
	settings.Numbers = false
	settings.Special = false

	if err := settings.Validate(); err == nil {
		t.Fatal("Validate() returned nil, want error")
	}
}

func assertWordList(t *testing.T, name string, words []string, minLength int) {
	t.Helper()

	if len(words) < minLength {
		t.Fatalf("%s word list has %d words, want at least %d", name, len(words), minLength)
	}

	seen := map[string]bool{}
	for _, word := range words {
		if word == "" {
			t.Fatalf("%s word list contains an empty word", name)
		}
		if !isPlainASCII(word) {
			t.Fatalf("%s word %q is not plain ASCII", name, word)
		}
		if word != strings.ToLower(word) {
			t.Fatalf("%s word %q is not lowercase", name, word)
		}
		if seen[word] {
			t.Fatalf("%s word list contains duplicate word %q", name, word)
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

func mustWordIndex(t *testing.T, words []string, want string) int {
	t.Helper()
	for i, word := range words {
		if word == want {
			return i
		}
	}
	t.Fatalf("word %q was not found", want)
	return 0
}

func fixedInts(values ...int) RandomIntFunc {
	i := 0
	return func(maxExclusive int) (int, error) {
		if maxExclusive <= 0 {
			return 0, fmt.Errorf("maxExclusive must be positive")
		}
		if i >= len(values) {
			return 0, fmt.Errorf("fixed int sequence exhausted")
		}
		value := values[i]
		i++
		if value < 0 {
			return 0, fmt.Errorf("fixed int %d is negative", value)
		}
		return value % maxExclusive, nil
	}
}

func assertMatches(t *testing.T, value, pattern string) {
	t.Helper()
	if !regexp.MustCompile(pattern).MatchString(value) {
		t.Fatalf("%q does not match %s", value, pattern)
	}
}
