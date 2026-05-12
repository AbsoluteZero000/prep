package resume

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ledongthuc/pdf"
)

func parsePDF(path string) (string, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening PDF: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	for i := 1; i <= r.NumPage(); i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, err := p.GetPlainText(nil)
		if err != nil {
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n")
	}

	extracted := buf.String()
	cleaned := cleanPDFText(extracted)

	// Detect scanned PDF: multi-page document with virtually no text
	if r.NumPage() > 1 && len(strings.Fields(cleaned)) < 20 {
		if tesseractAvailable() {
			ocrText, err := ocrWithTesseract(path)
			if err == nil {
				return ocrText, nil
			}
		}
		return "", ErrUnreadableContent
	}

	return cleaned, nil
}

func cleanPDFText(text string) string {
	// Normalize ligatures
	replacer := strings.NewReplacer(
		"ﬁ", "fi", "ﬂ", "fl", "ﬀ", "ff", "ﬃ", "ffi", "ﬄ", "ffl",
	)
	text = replacer.Replace(text)

	// Collapse 3+ consecutive newlines to 2
	for strings.Contains(text, "\n\n\n") {
		text = strings.ReplaceAll(text, "\n\n\n", "\n\n")
	}

	// Remove non-printable chars except \n and \t
	var cleaned strings.Builder
	for _, r := range text {
		if r >= 32 || r == '\n' || r == '\t' {
			cleaned.WriteRune(r)
		}
	}
	text = cleaned.String()

	// Trim leading/trailing whitespace per line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.Join(lines, "\n")
}

func tesseractAvailable() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}

func ocrWithTesseract(path string) (string, error) {
	cmd := exec.Command("tesseract", path, "stdout")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tesseract OCR failed: %w", err)
	}
	return out.String(), nil
}
