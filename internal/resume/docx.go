package resume

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

func parseDOCX(path string) (string, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return "", fmt.Errorf("opening DOCX archive: %w", err)
	}
	defer func() { _ = r.Close() }()

	var documentXML []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("opening document.xml: %w", err)
			}
			documentXML, err = io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return "", fmt.Errorf("reading document.xml: %w", err)
			}
			break
		}
	}

	if documentXML == nil {
		return "", ErrUnreadableContent
	}

	text := extractDocxText(string(documentXML))
	return text, nil
}

//nolint:gocyclo
func extractDocxText(xmlContent string) string {
	var buf bytes.Buffer

	// First pass: replace paragraph boundaries with newlines
	remaining := xmlContent

	// Track whether we are inside a <w:p> block
	inParagraph := false

	for {
		// Find the next relevant tag
		pOpen := strings.Index(remaining, "<w:p>")
		pClose := strings.Index(remaining, "</w:p>")
		tOpen := strings.Index(remaining, "<w:t")
		tClose := strings.Index(remaining, "</w:t>")

		// Find the earliest occurring tag
		type tagType int
		const (
			tagNone tagType = iota
			tagPOpen
			tagPClose
			tagTOpen
			tagTClose
		)

		var earliest tagType
		earliestPos := len(remaining)

		type candidate struct {
			pos int
			tag tagType
		}
		candidates := []candidate{
			{pOpen, tagPOpen},
			{pClose, tagPClose},
			{tOpen, tagTOpen},
			{tClose, tagTClose},
		}
		for _, c := range candidates {
			if c.pos >= 0 && c.pos < earliestPos {
				earliestPos = c.pos
				earliest = c.tag
			}
		}

		if earliest == tagNone {
			break
		}

		switch earliest {
		case tagPOpen:
			if inParagraph {
				buf.WriteByte('\n')
			}
			inParagraph = true
			remaining = remaining[earliestPos+5:]
		case tagPClose:
			if inParagraph {
				buf.WriteByte('\n')
			}
			inParagraph = false
			remaining = remaining[earliestPos+6:]
		case tagTOpen:
			// Find the closing > of the <w:t ...> tag
			closeBrace := strings.Index(remaining[earliestPos:], ">")
			if closeBrace == -1 {
				remaining = remaining[earliestPos+4:]
				continue
			}
			closeBrace += earliestPos
			// Find </w:t>
			endTag := strings.Index(remaining[closeBrace:], "</w:t>")
			if endTag == -1 {
				remaining = remaining[closeBrace+1:]
				continue
			}
			endTag += closeBrace

			content := remaining[closeBrace+1 : endTag]
			content = strings.ReplaceAll(content, "&amp;", "&")
			content = strings.ReplaceAll(content, "&lt;", "<")
			content = strings.ReplaceAll(content, "&gt;", ">")
			content = strings.ReplaceAll(content, "&quot;", "\"")
			content = strings.ReplaceAll(content, "&apos;", "'")

			if buf.Len() > 0 {
				b := buf.Bytes()
				if b[len(b)-1] != '\n' && b[len(b)-1] != ' ' {
					buf.WriteByte(' ')
				}
			}
			buf.WriteString(content)

			remaining = remaining[endTag+6:]
		case tagTClose:
			// Just advance past it
			remaining = remaining[earliestPos+6:]
		}
	}

	result := strings.TrimSpace(buf.String())
	return result
}
