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
	// MaxLength is the default fixed length for random passwords.
	MaxLength = 21

	MinAllowedLength       = 4
	RandomMinAllowedLength = 1
	MaxAllowedLength       = 128
)

type RandomIntFunc func(maxExclusive int) (int, error)

type Mode string

const (
	ModePassphrase Mode = "passphrase"
	ModeRandom     Mode = "random"
)

type Language string

const (
	LanguageCzech      Language = "czech"
	LanguageDanish     Language = "danish"
	LanguageDutch      Language = "dutch"
	LanguageEnglish    Language = "english"
	LanguageFinnish    Language = "finnish"
	LanguageFrench     Language = "french"
	LanguageGerman     Language = "german"
	LanguageHungarian  Language = "hungarian"
	LanguageItalian    Language = "italian"
	LanguageNorwegian  Language = "norwegian"
	LanguagePolish     Language = "polish"
	LanguagePortuguese Language = "portuguese"
	LanguageRomanian   Language = "romanian"
	LanguageSpanish    Language = "spanish"
	LanguageSwedish    Language = "swedish"
)

type LanguageOption struct {
	Language Language
	Label    string
}

var supportedLanguages = []LanguageOption{
	{Language: LanguageCzech, Label: "Czech"},
	{Language: LanguageDanish, Label: "Danish"},
	{Language: LanguageDutch, Label: "Dutch"},
	{Language: LanguageEnglish, Label: "English"},
	{Language: LanguageFinnish, Label: "Finnish"},
	{Language: LanguageFrench, Label: "French"},
	{Language: LanguageGerman, Label: "German"},
	{Language: LanguageHungarian, Label: "Hungarian"},
	{Language: LanguageItalian, Label: "Italian"},
	{Language: LanguageNorwegian, Label: "Norwegian"},
	{Language: LanguagePolish, Label: "Polish"},
	{Language: LanguagePortuguese, Label: "Portuguese"},
	{Language: LanguageRomanian, Label: "Romanian"},
	{Language: LanguageSpanish, Label: "Spanish"},
	{Language: LanguageSwedish, Label: "Swedish"},
}

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
	if !isSupportedLanguage(s.Language) {
		return fmt.Errorf("unsupported passphrase language %q", s.Language)
	}
	switch s.Mode {
	case ModePassphrase:
		if s.MinLength < MinAllowedLength {
			return fmt.Errorf("minimum length must be at least %d", MinAllowedLength)
		}
		if s.MinLength > MaxAllowedLength {
			return fmt.Errorf("minimum length must be at most %d", MaxAllowedLength)
		}
		if !s.Lowercase && !s.Uppercase {
			return fmt.Errorf("passphrases need lowercase, uppercase, or both enabled")
		}
		if len(wordsForLanguage(s.Language)) < 2 {
			return fmt.Errorf("passphrase word list must contain at least two words")
		}
	case ModeRandom:
		if s.MinLength < RandomMinAllowedLength {
			return fmt.Errorf("minimum length must be at least %d", RandomMinAllowedLength)
		}
		if s.MaxLength > MaxAllowedLength {
			return fmt.Errorf("maximum length must be at most %d", MaxAllowedLength)
		}
		if s.MinLength > s.MaxLength {
			return fmt.Errorf("minimum length must be less than or equal to maximum length")
		}
		groups := randomCharacterGroups(s)
		if len(groups) == 0 {
			return fmt.Errorf("random passwords need at least one character group enabled")
		}
	}

	return nil
}

func SupportedLanguages() []LanguageOption {
	languages := make([]LanguageOption, len(supportedLanguages))
	copy(languages, supportedLanguages)
	return languages
}

func LabelForLanguage(language Language) string {
	for _, option := range supportedLanguages {
		if option.Language == language {
			return option.Label
		}
	}
	return LabelForLanguage(LanguageNorwegian)
}

func LanguageForLabel(label string) (Language, bool) {
	for _, option := range supportedLanguages {
		if option.Label == label {
			return option.Language, true
		}
	}
	return "", false
}

// Generate returns a password like Fjell-Ovenfor3.
// It uses only words in norwegianWords, avoids æ/ø/å, and capitalizes the
// first letter of each word.
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
	wordCount, err := passphraseWordCount(settings, words)
	if err != nil {
		return "", err
	}

	separator := passphraseSeparator(settings)
	separatorLen := len(separator)
	numberLen := 0
	if settings.Numbers {
		numberLen = 1
	}

	minWordLengthSum := settings.MinLength - ((wordCount - 1) * separatorLen) - numberLen
	remainingSums := possibleLengthSums(words, wordCount)

	parts := make([]string, wordCount)
	usedLength := 0
	for i := range parts {
		remainingWords := wordCount - i - 1
		candidates := make([]string, 0, len(words))
		for _, word := range words {
			nextUsedLength := usedLength + len(word)
			if hasLengthSumAtLeast(remainingSums[remainingWords], minWordLengthSum-nextUsedLength) {
				candidates = append(candidates, word)
			}
		}
		if len(candidates) == 0 {
			return "", fmt.Errorf("could not generate a passphrase at least %d characters long", settings.MinLength)
		}

		idx, err := nextInt(len(candidates))
		if err != nil {
			return "", err
		}
		word := candidates[idx]
		parts[i] = applyWordCase(word, settings)
		usedLength += len(word)
	}

	pw := strings.Join(parts, separator)
	if settings.Numbers {
		digit, err := nextInt(10)
		if err != nil {
			return "", err
		}
		pw += strconv.Itoa(digit)
	}
	if len(pw) < settings.MinLength || !isPlainASCII(pw) {
		return "", fmt.Errorf("could not generate a passphrase at least %d characters long", settings.MinLength)
	}
	return pw, nil
}

func possibleLengthSums(words []string, maxWordCount int) []map[int]bool {
	wordLengths := uniqueWordLengths(words)
	sums := make([]map[int]bool, maxWordCount+1)
	sums[0] = map[int]bool{0: true}
	for count := 1; count <= maxWordCount; count++ {
		sums[count] = map[int]bool{}
		for previous := range sums[count-1] {
			for _, length := range wordLengths {
				sums[count][previous+length] = true
			}
		}
	}
	return sums
}

func uniqueWordLengths(words []string) []int {
	seen := map[int]bool{}
	lengths := make([]int, 0)
	for _, word := range words {
		length := len(word)
		if !seen[length] {
			seen[length] = true
			lengths = append(lengths, length)
		}
	}
	return lengths
}

func hasLengthSumAtLeast(sums map[int]bool, minSum int) bool {
	for sum := range sums {
		if sum >= minSum {
			return true
		}
	}
	return false
}

func generateRandomWithRand(settings Settings, nextInt RandomIntFunc) (string, error) {
	groups := randomCharacterGroups(settings)
	length, err := randBetween(nextInt, settings.MinLength, settings.MaxLength)
	if err != nil {
		return "", err
	}

	chars := make([]byte, 0, length)
	if length >= len(groups) {
		for _, group := range groups {
			idx, err := nextInt(len(group))
			if err != nil {
				return "", err
			}
			chars = append(chars, group[idx])
		}
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

func passphraseWordCount(settings Settings, words []string) (int, error) {
	_, longest := wordLengthRange(words)
	separatorLen := len(passphraseSeparator(settings))
	numberLen := 0
	if settings.Numbers {
		numberLen = 1
	}

	for wordCount := 2; wordCount <= 24; wordCount++ {
		if passphraseLengthForWords(wordCount, longest, separatorLen, numberLen) >= settings.MinLength {
			return wordCount, nil
		}
	}

	return 0, fmt.Errorf("settings cannot create a long enough passphrase")
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
	switch language {
	case LanguageCzech:
		return czechWords
	case LanguageDanish:
		return danishWords
	case LanguageDutch:
		return dutchWords
	case LanguageEnglish:
		return englishWords
	case LanguageFinnish:
		return finnishWords
	case LanguageFrench:
		return frenchWords
	case LanguageGerman:
		return germanWords
	case LanguageHungarian:
		return hungarianWords
	case LanguageItalian:
		return italianWords
	case LanguageNorwegian:
		return norwegianWords
	case LanguagePolish:
		return polishWords
	case LanguagePortuguese:
		return portugueseWords
	case LanguageRomanian:
		return romanianWords
	case LanguageSpanish:
		return spanishWords
	case LanguageSwedish:
		return swedishWords
	default:
		return nil
	}
}

func isSupportedLanguage(language Language) bool {
	return wordsForLanguage(language) != nil
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
