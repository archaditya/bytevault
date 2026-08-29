// Package textextract provides document text extraction for full-text search indexing.
// Supports PDF (via pdftotext subprocess), DOCX (ZIP/XML parsing), and plain text formats.
// All extraction is capped at MaxExtractBytes to prevent excessive storage usage.
package textextract

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/archaditya/bytevault/internal/logger"
)

// MaxExtractBytes is the maximum number of bytes of text to extract per file.
// This prevents oversized content_text entries in the database.
const MaxExtractBytes = 100 * 1024 // 100KB

// Extract reads the file content and extracts searchable text based on content type.
// Returns empty string (not error) for unsupported formats — this is intentional
// since most file types (images, video, archives) don't contain extractable text.
//
// The caller must provide a temporary file path where the content has already been
// written to disk. This is required because PDF and DOCX processing need random access (seek).
func ExtractFromFile(tmpPath string, contentType string, filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	ct := strings.ToLower(contentType)

	switch {
	case ct == "application/pdf" || ext == ".pdf":
		return extractPDF(tmpPath)

	case isDocx(ct, ext):
		return extractDOCX(tmpPath)

	case isPlainText(ct, ext):
		return extractPlainText(tmpPath)

	default:
		return "", nil // Unsupported format — not an error
	}
}

// extractPDF uses the pdftotext utility (from Poppler) to extract text from PDF files.
// Falls back gracefully if pdftotext is not available — the file simply won't have content search.
func extractPDF(tmpPath string) (string, error) {
	pdftotext, err := exec.LookPath("pdftotext")
	if err != nil {
		logger.Log.Warn().Msg("pdftotext not found. PDF content search will be unavailable. Install Poppler to enable.")
		return "", nil
	}

	// pdftotext <input.pdf> - (dash means output to stdout)
	cmd := exec.Command(pdftotext, "-q", "-enc", "UTF-8", tmpPath, "-")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logger.Log.Warn().Err(err).Str("stderr", stderr.String()).Msg("pdftotext extraction failed")
		return "", nil // Non-fatal: file will still be indexed by filename
	}

	return truncateText(stdout.String()), nil
}

// extractDOCX extracts text from .docx files by parsing the word/document.xml inside the ZIP container.
// Uses only Go stdlib — no external dependencies needed.
func extractDOCX(tmpPath string) (string, error) {
	r, err := zip.OpenReader(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to open docx as zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("failed to open word/document.xml: %w", err)
		}
		defer rc.Close()

		text, err := stripXMLToText(rc)
		if err != nil {
			return "", fmt.Errorf("failed to parse document.xml: %w", err)
		}

		return truncateText(text), nil
	}

	return "", nil // No document.xml found — unusual but not an error
}

// extractPlainText reads the first MaxExtractBytes from a plain text file.
func extractPlainText(tmpPath string) (string, error) {
	f, err := os.Open(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to open text file: %w", err)
	}
	defer f.Close()

	buf := make([]byte, MaxExtractBytes)
	n, err := io.ReadFull(f, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", fmt.Errorf("failed to read text file: %w", err)
	}

	return string(buf[:n]), nil
}

// stripXMLToText extracts text content from an XML stream, stripping all tags.
// Used for DOCX word/document.xml parsing — collects text from <w:t> elements.
func stripXMLToText(r io.Reader) (string, error) {
	decoder := xml.NewDecoder(r)
	var sb strings.Builder
	var inTextElement bool

	for {
		tok, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return sb.String(), nil // Return what we have on parse error
		}

		switch t := tok.(type) {
		case xml.StartElement:
			// <w:t> elements contain the actual text in DOCX
			if t.Name.Local == "t" {
				inTextElement = true
			}
			// <w:p> elements represent paragraphs — add newline between them
			if t.Name.Local == "p" && sb.Len() > 0 {
				sb.WriteString("\n")
			}
		case xml.EndElement:
			if t.Name.Local == "t" {
				inTextElement = false
			}
		case xml.CharData:
			if inTextElement {
				sb.Write(t)
			}
		}

		// Safety cap: stop parsing if we've exceeded the max
		if sb.Len() >= MaxExtractBytes {
			break
		}
	}

	return sb.String(), nil
}

// truncateText ensures the text doesn't exceed MaxExtractBytes.
// Truncates at a UTF-8 safe boundary.
func truncateText(s string) string {
	if len(s) <= MaxExtractBytes {
		return strings.TrimSpace(s)
	}
	// Truncate and find the last space to avoid cutting mid-word
	truncated := s[:MaxExtractBytes]
	if lastSpace := strings.LastIndex(truncated, " "); lastSpace > MaxExtractBytes-200 {
		truncated = truncated[:lastSpace]
	}
	return strings.TrimSpace(truncated)
}

// isDocx checks if the content type or extension indicates a DOCX file.
func isDocx(contentType, ext string) bool {
	return ext == ".docx" ||
		contentType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
}

// isPlainText checks if the content type or extension indicates a plain text file
// that can be directly read for search indexing.
func isPlainText(contentType, ext string) bool {
	// Explicit text/* MIME types
	if strings.HasPrefix(contentType, "text/") {
		return true
	}

	// JSON, YAML, and other structured text formats
	textMimeTypes := map[string]bool{
		"application/json":                true,
		"application/xml":                 true,
		"application/javascript":          true,
		"application/x-yaml":             true,
		"application/yaml":               true,
		"application/toml":               true,
		"application/x-sh":               true,
		"application/x-shellscript":      true,
		"application/graphql":            true,
		"application/sql":                true,
		"application/x-httpd-php":        true,
		"application/x-python-code":      true,
		"application/typescript":         true,
		"application/rtf":                true,
	}
	if textMimeTypes[contentType] {
		return true
	}

	// Fallback: known text file extensions
	textExtensions := map[string]bool{
		".txt": true, ".md": true, ".csv": true, ".json": true,
		".yaml": true, ".yml": true, ".toml": true, ".xml": true,
		".html": true, ".htm": true, ".css": true, ".js": true,
		".ts": true, ".tsx": true, ".jsx": true,
		".go": true, ".py": true, ".rb": true, ".rs": true,
		".java": true, ".c": true, ".cpp": true, ".h": true,
		".sh": true, ".bash": true, ".zsh": true,
		".sql": true, ".graphql": true, ".gql": true,
		".env": true, ".ini": true, ".cfg": true, ".conf": true,
		".log": true, ".rtf": true,
		".svelte": true, ".vue": true, ".php": true,
		".swift": true, ".kt": true, ".kts": true,
		".dart": true, ".lua": true, ".r": true,
		".dockerfile": true, ".makefile": true,
	}
	return textExtensions[ext]
}
