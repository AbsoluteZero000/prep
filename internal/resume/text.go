package resume

import (
	"fmt"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

func parseText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading text file: %w", err)
	}

	if len(data) == 0 {
		return "", ErrEmptyResume
	}

	// Try UTF-8 first
	if utf8.Valid(data) {
		text := string(data)
		text = normalizeLineEndings(text)
		return strings.TrimSpace(text), nil
	}

	// Attempt Latin-1 (ISO-8859-1) to UTF-8 conversion
	decoder := charmap.ISO8859_1.NewDecoder()
	decoded, err := decoder.Bytes(data)
	if err != nil {
		return "", fmt.Errorf("decoding text as Latin-1: %w", err)
	}
	text := string(decoded)
	text = normalizeLineEndings(text)
	return strings.TrimSpace(text), nil
}

func normalizeLineEndings(text string) string {
	// Replace \r\n and \r with \n
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}
