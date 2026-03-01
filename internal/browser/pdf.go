package browser

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-rod/rod/lib/proto"
)

// PDFOptions represents PDF export options
type PDFOptions struct {
	Scale         float64 `json:"scale,omitempty"`
	PrintBackground bool   `json:"print_background,omitempty"`
	Landscape     bool   `json:"landscape,omitempty"`
	Format        string  `json:"format,omitempty"` // A4, Letter, etc.
	Margins       Margin  `json:"margins,omitempty"`
	PageRanges    string  `json:"page_ranges,omitempty"` // "1-5, 8, 11-13"
	HeaderHTML    string  `json:"header_html,omitempty"`
	FooterHTML    string  `json:"footer_html,omitempty"`
}

// Margin represents page margins
type Margin struct {
	Top    string `json:"top,omitempty"`
	Bottom string `json:"bottom,omitempty"`
	Left   string `json:"left,omitempty"`
	Right  string `json:"right,omitempty"`
}

// ExportPDF exports the page as PDF
func (s *Session) ExportPDF(path string, opts *PDFOptions) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if opts == nil {
		opts = &PDFOptions{}
	}

	// Validate path
	if !strings.HasSuffix(path, ".pdf") {
		path = path + ".pdf"
	}

	// Get PDF with options (simplified to default for now)
	reader, err := s.Page.PDF(&proto.PagePrintToPDF{})
	if err != nil {
		return fmt.Errorf("failed to generate PDF: %w", err)
	}
	defer reader.Close()

	// Create file
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create PDF file: %w", err)
	}
	defer file.Close()

	// Write to file
	_, err = io.Copy(file, reader)
	if err != nil {
		return fmt.Errorf("failed to write PDF: %w", err)
	}

	return nil
}
