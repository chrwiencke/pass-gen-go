package password

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

const (
	// MinLength is 15 because the password must be over 14 characters long.
	MinLength = 15
	// MaxLength is 21 because the password must be under 22 characters long.
	MaxLength = 21
)

type RandomIntFunc func(maxExclusive int) (int, error)

// Generate returns a password like Fjell-Ovenfor3.
// It uses only words in norwegianWords, avoids æ/ø/å, capitalizes the first
// letter of each word, and retries until the final string is over 14 and under
// 22 characters long.
func Generate() (string, error) {
	return generateWithRand(cryptoRandInt)
}

func generateWithRand(nextInt RandomIntFunc) (string, error) {
	if len(norwegianWords) < 2 {
		return "", fmt.Errorf("word list must contain at least two words")
	}

	for attempts := 0; attempts < 1000; attempts++ {
		firstIdx, err := nextInt(len(norwegianWords))
		if err != nil {
			return "", err
		}

		secondIdx, err := nextInt(len(norwegianWords))
		if err != nil {
			return "", err
		}

		digit, err := nextInt(10)
		if err != nil {
			return "", err
		}

		first := capitalizeWord(norwegianWords[firstIdx])
		second := capitalizeWord(norwegianWords[secondIdx])
		pw := fmt.Sprintf("%s-%s%d", first, second, digit)
		if len(pw) >= MinLength && len(pw) <= MaxLength && isPlainASCII(pw) {
			return pw, nil
		}
	}

	return "", fmt.Errorf("could not generate a password between %d and %d characters", MinLength, MaxLength)
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
