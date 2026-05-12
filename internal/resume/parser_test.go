package resume

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectFormat_PDF(t *testing.T) {
	fmt_, err := DetectFormat("testdata/resumes/sample.pdf")
	if err != ErrFileNotFound {
		t.Fatalf("expected ErrFileNotFound for missing sample.pdf, got %v", err)
	}

	fmt_, err = DetectFormat("testdata/resumes/empty.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if fmt_ != FormatPDF {
		t.Fatalf("expected FormatPDF, got %v", fmt_)
	}
}

func TestDetectFormat_DOCX(t *testing.T) {
	fmt_, err := DetectFormat("testdata/resumes/sample.docx")
	if err != nil {
		t.Fatal(err)
	}
	if fmt_ != FormatDOCX {
		t.Fatalf("expected FormatDOCX, got %v", fmt_)
	}
}

func TestDetectFormat_Text(t *testing.T) {
	fmt_, err := DetectFormat("testdata/resumes/sample.txt")
	if err != nil {
		t.Fatal(err)
	}
	if fmt_ != FormatText {
		t.Fatalf("expected FormatText, got %v", fmt_)
	}
}

func TestDetectFormat_Unsupported(t *testing.T) {
	_, err := DetectFormat("testdata/resumes/nonexistent.xyz")
	if err != ErrUnsupportedFormat && err != ErrFileNotFound {
		t.Fatalf("expected ErrUnsupportedFormat or ErrFileNotFound, got %v", err)
	}
}

func TestParse_SampleText(t *testing.T) {
	result, err := Parse("testdata/resumes/sample.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.WordCount < 50 {
		t.Fatalf("expected >= 50 words, got %d", result.WordCount)
	}
	if !strings.Contains(result.RawText, "Go") {
		t.Fatal("expected 'Go' in parsed text")
	}
	if result.Hash == "" {
		t.Fatal("expected non-empty hash")
	}
}

func TestParse_SampleDOCX(t *testing.T) {
	result, err := Parse("testdata/resumes/sample.docx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.WordCount < 10 {
		t.Fatalf("expected >= 10 words, got %d", result.WordCount)
	}
	if !strings.Contains(result.RawText, "John Doe") {
		t.Fatal("expected 'John Doe' in parsed text")
	}
}

func TestParse_EmptyPDF(t *testing.T) {
	_, err := Parse("testdata/resumes/empty.pdf")
	if err != ErrEmptyResume {
		t.Fatalf("expected ErrEmptyResume, got %v", err)
	}
}

func TestParse_ScannedPDF(t *testing.T) {
	_, err := Parse("testdata/resumes/scanned.pdf")
	if err != ErrUnreadableContent {
		t.Fatalf("expected ErrUnreadableContent, got %v", err)
	}
}

func TestParse_OversizedText(t *testing.T) {
	result, err := Parse("testdata/resumes/oversized.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TokenEstimate > 6000 {
		t.Fatalf("expected TokenEstimate <= 6000 after truncation, got %d", result.TokenEstimate)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "truncated") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected truncation warning")
	}
}

func TestParse_FileNotFound(t *testing.T) {
	_, err := Parse("testdata/resumes/nonexistent.pdf")
	if err != ErrFileNotFound {
		t.Fatalf("expected ErrFileNotFound, got %v", err)
	}
}

func TestParse_UnsupportedExtension(t *testing.T) {
	// Create a temp file with unsupported extension
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "resume.xyz")
	os.WriteFile(path, []byte("hello world"), 0644)

	_, err := Parse(path)
	if err != ErrUnsupportedFormat {
		t.Fatalf("expected ErrUnsupportedFormat, got %v", err)
	}
}

func TestParse_TextWithMarkdownContent(t *testing.T) {
	result, err := Parse("testdata/resumes/sample.txt")
	if err != nil {
		t.Fatal(err)
	}
	if result.Format != FormatText {
		t.Fatalf("expected FormatText, got %v", result.Format)
	}
}

func TestParse_ShortResumeReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "short.txt")
	os.WriteFile(path, []byte("Hello world."), 0644)

	_, err := Parse(path)
	if err != ErrResumeTooShort {
		t.Fatalf("expected ErrResumeTooShort for short resume, got %v", err)
	}
}

func TestParse_EmptyFileReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.txt")
	os.WriteFile(path, []byte{}, 0644)

	_, err := Parse(path)
	if err != ErrEmptyResume {
		t.Fatalf("expected ErrEmptyResume, got %v", err)
	}
}

func TestTruncateToTokenBudget(t *testing.T) {
	// Create a long text that would exceed budget
	text := strings.Repeat("This is a test sentence for truncation. ", 500)
	truncated := TruncateToTokenBudget(text, 100)
	if len(truncated) < len(text) {
		if !strings.Contains(truncated, "[Resume truncated") {
			t.Fatal("expected truncation marker")
		}
	} else {
		t.Fatal("expected truncation to reduce length")
	}
}

func TestParse_ResumeHeuristicWarning(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "notaresume.txt")
	os.WriteFile(path, []byte(strings.Repeat("Lorem ipsum dolor sit amet. ", 30)), 0644)

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("expected no error but heuristic warning, got %v", err)
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "does not appear to be a resume") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected ErrNotAResume warning in non-resume text")
	}
}
