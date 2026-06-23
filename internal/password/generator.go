package password

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

const (
	// MinLength is 15 because the password must be over 14 characters long.
	MinLength = 15
	// MaxLength is 21 because the password must be under 22 characters long.
	MaxLength = 21

	MinAllowedLength = 4
	MaxAllowedLength = 128
)

type RandomIntFunc func(maxExclusive int) (int, error)

type Mode string

const (
	ModePassphrase Mode = "passphrase"
	ModeRandom     Mode = "random"
)

type Language string

const (
	LanguageNorwegian Language = "norwegian"
	LanguageEnglish   Language = "english"
)

type Settings struct {
	Mode      Mode     `json:"mode"`
	Language  Language `json:"language"`
	MinLength int      `json:"minLength"`
	MaxLength int      `json:"maxLength"`
	Lowercase bool     `json:"lowercase"`
	Uppercase bool     `json:"uppercase"`
	Numbers   bool     `json:"numbers"`
	Special   bool     `json:"special"`
}

func DefaultSettings() Settings {
	return Settings{
		Mode:      ModePassphrase,
		Language:  LanguageNorwegian,
		MinLength: MinLength,
		MaxLength: MaxLength,
		Lowercase: true,
		Uppercase: true,
		Numbers:   true,
		Special:   true,
	}
}

func (s Settings) Normalize() Settings {
	if s.Mode == "" {
		s.Mode = ModePassphrase
	}
	if s.Language == "" {
		s.Language = LanguageNorwegian
	}
	if s.MinLength == 0 {
		s.MinLength = MinLength
	}
	if s.MaxLength == 0 {
		s.MaxLength = MaxLength
	}
	return s
}

func (s Settings) Validate() error {
	s = s.Normalize()

	if s.Mode != ModePassphrase && s.Mode != ModeRandom {
		return fmt.Errorf("unsupported password mode %q", s.Mode)
	}
	if s.Language != LanguageNorwegian && s.Language != LanguageEnglish {
		return fmt.Errorf("unsupported passphrase language %q", s.Language)
	}
	if s.MinLength < MinAllowedLength {
		return fmt.Errorf("minimum length must be at least %d", MinAllowedLength)
	}
	if s.MaxLength > MaxAllowedLength {
		return fmt.Errorf("maximum length must be at most %d", MaxAllowedLength)
	}
	if s.MinLength > s.MaxLength {
		return fmt.Errorf("minimum length must be less than or equal to maximum length")
	}

	switch s.Mode {
	case ModePassphrase:
		if !s.Lowercase && !s.Uppercase {
			return fmt.Errorf("passphrases need lowercase, uppercase, or both enabled")
		}
		if len(wordsForLanguage(s.Language)) < 2 {
			return fmt.Errorf("passphrase word list must contain at least two words")
		}
	case ModeRandom:
		groups := randomCharacterGroups(s)
		if len(groups) == 0 {
			return fmt.Errorf("random passwords need at least one character group enabled")
		}
		if len(groups) > s.MaxLength {
			return fmt.Errorf("maximum length must be at least %d for the selected character groups", len(groups))
		}
	}

	return nil
}

// Generate returns a password like Fjell-Ovenfor3.
// It uses only words in norwegianWords, avoids æ/ø/å, capitalizes the first
// letter of each word, and retries until the final string is over 14 and under
// 22 characters long.
func Generate() (string, error) {
	return generateWithRand(cryptoRandInt)
}

func GenerateWithSettings(settings Settings) (string, error) {
	return generateWithSettingsWithRand(settings, cryptoRandInt)
}

func generateWithRand(nextInt RandomIntFunc) (string, error) {
	return generateWithSettingsWithRand(DefaultSettings(), nextInt)
}

func generateWithSettingsWithRand(settings Settings, nextInt RandomIntFunc) (string, error) {
	settings = settings.Normalize()
	if err := settings.Validate(); err != nil {
		return "", err
	}

	if settings.Mode == ModeRandom {
		return generateRandomWithRand(settings, nextInt)
	}

	return generatePassphraseWithRand(settings, nextInt)
}

func generatePassphraseWithRand(settings Settings, nextInt RandomIntFunc) (string, error) {
	words := wordsForLanguage(settings.Language)
	minWords, maxWords, err := passphraseWordCountRange(settings, words)
	if err != nil {
		return "", err
	}

	for attempts := 0; attempts < 1000; attempts++ {
		wordCount, err := randBetween(nextInt, minWords, maxWords)
		if err != nil {
			return "", err
		}

		parts := make([]string, wordCount)
		for i := range parts {
			idx, err := nextInt(len(words))
			if err != nil {
				return "", err
			}

			parts[i] = applyWordCase(words[idx], settings)
		}

		pw := strings.Join(parts, passphraseSeparator(settings))
		if settings.Numbers {
			digit, err := nextInt(10)
			if err != nil {
				return "", err
			}
			pw += strconv.Itoa(digit)
		}
		if len(pw) >= settings.MinLength && len(pw) <= settings.MaxLength && isPlainASCII(pw) {
			return pw, nil
		}
	}

	return "", fmt.Errorf("could not generate a passphrase between %d and %d characters", settings.MinLength, settings.MaxLength)
}

func generateRandomWithRand(settings Settings, nextInt RandomIntFunc) (string, error) {
	groups := randomCharacterGroups(settings)
	minLength := settings.MinLength
	if minLength < len(groups) {
		minLength = len(groups)
	}

	length, err := randBetween(nextInt, minLength, settings.MaxLength)
	if err != nil {
		return "", err
	}

	chars := make([]byte, 0, length)
	for _, group := range groups {
		idx, err := nextInt(len(group))
		if err != nil {
			return "", err
		}
		chars = append(chars, group[idx])
	}

	all := strings.Join(groups, "")
	for len(chars) < length {
		idx, err := nextInt(len(all))
		if err != nil {
			return "", err
		}
		chars = append(chars, all[idx])
	}

	if err := shuffleBytes(chars, nextInt); err != nil {
		return "", err
	}

	return string(chars), nil
}

func passphraseWordCountRange(settings Settings, words []string) (int, int, error) {
	shortest, longest := wordLengthRange(words)
	separatorLen := len(passphraseSeparator(settings))
	numberLen := 0
	if settings.Numbers {
		numberLen = 1
	}

	maxAllowedWords := 2
	if settings.MaxLength > MaxLength {
		maxAllowedWords = 24
	}

	minWords := 2
	for minWords < maxAllowedWords && passphraseLengthForWords(minWords, longest, separatorLen, numberLen) < settings.MinLength {
		minWords++
	}
	if passphraseLengthForWords(minWords, longest, separatorLen, numberLen) < settings.MinLength {
		return 0, 0, fmt.Errorf("settings cannot create a long enough passphrase")
	}

	maxWords := maxAllowedWords
	for maxWords >= 2 && passphraseLengthForWords(maxWords, shortest, separatorLen, numberLen) > settings.MaxLength {
		maxWords--
	}
	if maxWords < 2 {
		return 0, 0, fmt.Errorf("settings cannot create a short enough passphrase")
	}
	if minWords > maxWords {
		return 0, 0, fmt.Errorf("settings cannot create a passphrase between %d and %d characters", settings.MinLength, settings.MaxLength)
	}

	return minWords, maxWords, nil
}

func passphraseLengthForWords(wordCount, wordLength, separatorLen, numberLen int) int {
	return wordCount*wordLength + (wordCount-1)*separatorLen + numberLen
}

func wordLengthRange(words []string) (int, int) {
	shortest := len(words[0])
	longest := len(words[0])
	for _, word := range words[1:] {
		if len(word) < shortest {
			shortest = len(word)
		}
		if len(word) > longest {
			longest = len(word)
		}
	}
	return shortest, longest
}

func passphraseSeparator(settings Settings) string {
	if settings.Special {
		return "-"
	}
	return ""
}

func applyWordCase(word string, settings Settings) string {
	switch {
	case settings.Lowercase && settings.Uppercase:
		return capitalizeWord(word)
	case settings.Uppercase:
		return strings.ToUpper(word)
	default:
		return strings.ToLower(word)
	}
}

func wordsForLanguage(language Language) []string {
	if language == LanguageEnglish {
		return englishWords
	}
	return norwegianWords
}

func randomCharacterGroups(settings Settings) []string {
	groups := make([]string, 0, 4)
	if settings.Lowercase {
		groups = append(groups, "abcdefghijklmnopqrstuvwxyz")
	}
	if settings.Uppercase {
		groups = append(groups, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	}
	if settings.Numbers {
		groups = append(groups, "0123456789")
	}
	if settings.Special {
		groups = append(groups, "!@#$%^&*_-+=?")
	}
	return groups
}

func randBetween(nextInt RandomIntFunc, minInclusive, maxInclusive int) (int, error) {
	if minInclusive > maxInclusive {
		return 0, fmt.Errorf("minimum value must be less than or equal to maximum value")
	}
	offset, err := nextInt(maxInclusive - minInclusive + 1)
	if err != nil {
		return 0, err
	}
	return minInclusive + offset, nil
}

func shuffleBytes(chars []byte, nextInt RandomIntFunc) error {
	for i := len(chars) - 1; i > 0; i-- {
		j, err := nextInt(i + 1)
		if err != nil {
			return err
		}
		chars[i], chars[j] = chars[j], chars[i]
	}
	return nil
}

func capitalizeWord(word string) string {
	if word == "" {
		return ""
	}
	return strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
}

func cryptoRandInt(maxExclusive int) (int, error) {
	if maxExclusive <= 0 {
		return 0, fmt.Errorf("maxExclusive must be positive")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxExclusive)))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

func isPlainASCII(s string) bool {
	if strings.ContainsAny(s, "æøåÆØÅ") {
		return false
	}
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}
