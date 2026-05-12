// Package resume parses resume files in PDF, DOCX, and plain text formats.
package resume

import (
	"crypto/sha256"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

// Format represents the detected file format of a resume.
type Format int

const (
	FormatPDF      Format = iota
	FormatDOCX     Format = iota
	FormatText     Format = iota
	FormatMarkdown Format = iota
	FormatUnknown  Format = iota
)

// ParseResult holds the extracted content and metadata from a resume file.
type ParseResult struct {
	RawText       string
	WordCount     int
	TokenEstimate int
	Format        Format
	Warnings      []string
	Hash          string
}

// DetectFormat identifies the resume format by extension and magic bytes.
func DetectFormat(path string) (Format, error) {
	f, err := os.Open(path) //nolint:gosec // user-provided path is intentional
	if err != nil {
		if os.IsNotExist(err) {
			return FormatUnknown, ErrFileNotFound
		}
		return FormatUnknown, fmt.Errorf("opening file: %w", err)
	}
	defer func() { _ = f.Close() }()

	header := make([]byte, 8)
	n, _ := f.Read(header)

	ext := strings.ToLower(filepath.Ext(path))

	// Check magic bytes first (stronger signal), but only if we read enough
	if n >= 5 {
		magic := string(header[:5])
		switch {
		case magic == "%PDF-":
			return FormatPDF, nil
		case n >= 2 && string(header[:2]) == "PK":
			if ext == ".docx" {
				return FormatDOCX, nil
			}
			return FormatUnknown, ErrUnsupportedFormat
		}
	}

	// Fall back to extension detection
	switch ext {
	case ".pdf":
		return FormatPDF, nil
	case ".docx":
		return FormatDOCX, nil
	case ".txt", ".text":
		return FormatText, nil
	case ".md":
		return FormatMarkdown, nil
	default:
		return FormatUnknown, ErrUnsupportedFormat
	}
}

// Parse reads a resume file, extracts text, and returns the result.
func Parse(path string) (ParseResult, error) {
	var result ParseResult

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, ErrFileNotFound
		}
		return result, fmt.Errorf("stat file: %w", err)
	}
	if info.Size() == 0 {
		return result, ErrEmptyResume
	}

	fmt_, err := DetectFormat(path)
	if err != nil {
		return result, err
	}

	var rawText string
	switch fmt_ {
	case FormatPDF:
		rawText, err = parsePDF(path)
	case FormatDOCX:
		rawText, err = parseDOCX(path)
	case FormatText, FormatMarkdown:
		rawText, err = parseText(path)
	default:
		return result, ErrUnsupportedFormat
	}
	if err != nil {
		return result, err
	}

	words := strings.Fields(rawText)
	wordCount := len(words)
	tokenEst := int(math.Ceil(float64(wordCount) * 1.35))

	result.RawText = rawText
	result.WordCount = wordCount
	result.TokenEstimate = tokenEst
	result.Format = fmt_

	// Post-parse validation
	if wordCount < 50 {
		return result, ErrResumeTooShort
	}

	// Heuristic: check for resume-like content
	resumeKeywords := []string{"experience", "education", "skills", "work"}
	lcText := strings.ToLower(rawText)
	matched := 0
	for _, kw := range resumeKeywords {
		if strings.Contains(lcText, kw) {
			matched++
		}
	}
	if matched < 2 {
		result.Warnings = append(result.Warnings, ErrNotAResume.Error())
	}

	// Truncate if over token budget
	if tokenEst > 6000 {
		result.RawText = TruncateToTokenBudget(rawText, 6000)
		result.TokenEstimate = 6000
		result.Warnings = append(result.Warnings, "resume truncated to fit context window")
	}

	// Compute hash
	h := sha256.Sum256([]byte(result.RawText))
	result.Hash = fmt.Sprintf("%x", h)

	return result, nil
}

// TruncateToTokenBudget trims text to fit within maxTokens by dropping sections
// from the bottom, preserving contact info, skills, and recent experience.
//
//nolint:gocyclo
func TruncateToTokenBudget(text string, maxTokens int) string {
	sections := strings.Split(text, "\n\n")
	if len(sections) <= 1 {
		// Single block — just truncate by words
		words := strings.Fields(text)
		wordLimit := int(float64(maxTokens) / 1.35)
		if len(words) <= wordLimit {
			return text
		}
		return strings.Join(words[:wordLimit], " ") + " [Resume truncated to fit context window]"
	}

	// Score sections to prioritize
	type scored struct {
		text  string
		score int
	}

	scoredSections := make([]scored, len(sections))
	lcSections := make([]string, len(sections))
	for i, s := range sections {
		lcSections[i] = strings.ToLower(s)
		score := 0
		// Contact info / header (usually first section)
		if i == 0 {
			score += 10
		}
		if strings.Contains(lcSections[i], "skills") {
			score += 8
		}
		if strings.Contains(lcSections[i], "experience") || strings.Contains(lcSections[i], "work") {
			score += 6
		}
		if strings.Contains(lcSections[i], "education") {
			score += 4
		}
		// Recent sections get higher base score (reverse index)
		score += i
		scoredSections[i] = scored{text: s, score: score}
	}

	// Bubble sort by score ascending (lowest score first = most droppable)
	for i := 0; i < len(scoredSections); i++ {
		for j := i + 1; j < len(scoredSections); j++ {
			if scoredSections[i].score > scoredSections[j].score {
				scoredSections[i], scoredSections[j] = scoredSections[j], scoredSections[i]
			}
		}
	}

	// Build result from highest-scored sections up
	var result []string
	wordLimit := int(float64(maxTokens) / 1.35)
	wordCount := 0

	// Process from highest score to lowest
	for i := len(scoredSections) - 1; i >= 0; i-- {
		secWords := len(strings.Fields(scoredSections[i].text))
		if wordCount+secWords <= wordLimit {
			result = append(result, scoredSections[i].text)
			wordCount += secWords
		}
	}

	if len(result) == 0 {
		// Shouldn't happen with valid input, but fallback
		words := strings.Fields(text)
		if len(words) > wordLimit {
			words = words[:wordLimit]
		}
		return strings.Join(words, " ") + " [Resume truncated to fit context window]"
	}

	return strings.Join(result, "\n\n") + "\n\n[Resume truncated to fit context window]"
}
