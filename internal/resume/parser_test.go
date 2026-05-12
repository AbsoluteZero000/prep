package resume

import (
	"archive/zip"
	"bytes"
	"fmt"
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

func TestParse_PDFWithContent(t *testing.T) {
	path := writeTestPDF(t, resumeContent)
	result, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatPDF {
		t.Fatalf("expected FormatPDF, got %v", result.Format)
	}
	if !strings.Contains(result.RawText, "Ahmed Wael") {
		t.Fatalf("expected 'Ahmed Wael' in parsed text, got:\n%s", result.RawText)
	}
	if !strings.Contains(result.RawText, "Software Engineer") {
		t.Fatalf("expected 'Software Engineer' in parsed text, got:\n%s", result.RawText)
	}
	if !strings.Contains(result.RawText, "Cairo University") {
		t.Fatalf("expected 'Cairo University' in parsed text, got:\n%s", result.RawText)
	}
	if result.Hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if len(result.Warnings) > 0 {
		t.Fatalf("unexpected warnings: %v", result.Warnings)
	}
}

// resumeContent is a realistic resume text used for PDF extraction tests.
// It must be long enough (>50 words) to pass heuristic validation.
const resumeContent = `Ahmed Wael Wanas
Software Engineer
Giza Egypt

SUMMARY
Backend Software Engineer specializing in building high-performance scalable distributed systems using Java Spring Boot Go and Python. Experienced in microservices architecture performance optimization and production-grade backend systems.

EXPERIENCE
Adres Amman Jordan
Software Engineer Dec 2025 Present
Contributed to the development and maintenance of a production-grade real estate platform for Sharjah. Wrote and maintained high-quality scalable Java Spring Boot code in a real-world enterprise environment. Led performance improvements through Spring framework upgrades and targeted code refactoring. Integrated and managed database migrations using Flyway ensuring smooth version control and deployment.

EDUCATION
Cairo University
BCS Software Engineering Oct 2020 Jul 2024
Grade Excellent with honors

SKILLS
Java Go Python Spring Boot FastAPI Docker Kubernetes PostgreSQL Redis AWS Microservices REST APIs`

func TestParse_DOCXWithContent(t *testing.T) {
	path := writeTestDOCX(t, resumeContent)
	result, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatDOCX {
		t.Fatalf("expected FormatDOCX, got %v", result.Format)
	}
	if !strings.Contains(result.RawText, "Ahmed Wael") {
		t.Fatalf("expected 'Ahmed Wael' in parsed text, got:\n%s", result.RawText)
	}
	if !strings.Contains(result.RawText, "Software Engineer") {
		t.Fatalf("expected 'Software Engineer' in parsed text, got:\n%s", result.RawText)
	}
	if !strings.Contains(result.RawText, "Cairo University") {
		t.Fatalf("expected 'Cairo University' in parsed text, got:\n%s", result.RawText)
	}
	if result.Hash == "" {
		t.Fatal("expected non-empty hash")
	}
}

func TestParse_TXTWithContent(t *testing.T) {
	path := writeTestTXT(t, resumeContent)
	result, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Format != FormatText {
		t.Fatalf("expected FormatText, got %v", result.Format)
	}
	if !strings.Contains(result.RawText, "Ahmed Wael") {
		t.Fatalf("expected 'Ahmed Wael' in parsed text, got:\n%s", result.RawText)
	}
	if !strings.Contains(result.RawText, "Software Engineer") {
		t.Fatalf("expected 'Software Engineer' in parsed text, got:\n%s", result.RawText)
	}
	if !strings.Contains(result.RawText, "Cairo University") {
		t.Fatalf("expected 'Cairo University' in parsed text, got:\n%s", result.RawText)
	}
	if result.Hash == "" {
		t.Fatal("expected non-empty hash")
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

// writeTestPDF generates a minimal valid PDF with the given text content
// and returns its path. The text must use only ASCII printable characters.
func writeTestPDF(t *testing.T, text string) string {
	t.Helper()

	content := fmt.Sprintf(`BT /F1 12 Tf 100 700 Td (%s) Tj ET`, text)
	streamLen := len(content)

	var buf bytes.Buffer
	offsets := make([]int, 6)

	buf.WriteString("%PDF-1.4\n")

	offsets[1] = buf.Len()
	buf.WriteString("1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n")

	offsets[2] = buf.Len()
	buf.WriteString("2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n")

	offsets[3] = buf.Len()
	buf.WriteString("3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 612 792]/Contents 4 0 R/Resources<</Font<</F1 5 0 R>>>>>>endobj\n")

	offsets[4] = buf.Len()
	buf.WriteString(fmt.Sprintf("4 0 obj<</Length %d>>stream\n%s\nendstream\nendobj\n", streamLen, content))

	offsets[5] = buf.Len()
	buf.WriteString("5 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\n")

	xrefOffset := buf.Len()
	buf.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString("trailer\n<</Size 6/Root 1 0 R>>\n")
	buf.WriteString(fmt.Sprintf("startxref\n%d\n%%%%EOF\n", xrefOffset))

	path := filepath.Join(t.TempDir(), "resume.pdf")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeTestDOCX creates a minimal DOCX file with the given text content
// and returns its path. Each paragraph in the output is separated by a
// newline as the DOCX parser treats <w:p> boundaries as line breaks.
func writeTestDOCX(t *testing.T, text string) string {
	t.Helper()

	var docXML bytes.Buffer
	docXML.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	docXML.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">`)
	docXML.WriteString("<w:body>")
	for _, line := range strings.Split(text, "\n") {
		docXML.WriteString("<w:p><w:r><w:t>")
		docXML.WriteString(escapeXML(line))
		docXML.WriteString("</w:t></w:r></w:p>")
	}
	docXML.WriteString("</w:body></w:document>")

	path := filepath.Join(t.TempDir(), "resume.docx")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(docXML.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// writeTestTXT creates a plain text file with the given content
// and returns its path.
func writeTestTXT(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "resume.txt")
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// escapeXML escapes special XML characters in a string.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
