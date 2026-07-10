package tools

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/sipeed/jameclaw/pkg/config"
	"github.com/sipeed/jameclaw/pkg/media"
)

const (
	pdfPageWidth     = 612
	pdfPageHeight    = 792
	pdfMargin        = 54
	pdfFontSize      = 11
	pdfLineHeight    = 15
	pdfMaxLineRunes  = 86
	defaultPDFSubdir = "generated"
)

// CreatePDFTool creates a simple text PDF and sends it through the media pipeline.
type CreatePDFTool struct {
	workspace   string
	restrict    bool
	maxFileSize int
	mediaStore  media.MediaStore
	allowPaths  []*regexp.Regexp
}

func NewCreatePDFTool(
	workspace string,
	restrict bool,
	maxFileSize int,
	store media.MediaStore,
	allowPaths ...[]*regexp.Regexp,
) *CreatePDFTool {
	if maxFileSize <= 0 {
		maxFileSize = config.DefaultMaxMediaSize
	}
	var patterns []*regexp.Regexp
	if len(allowPaths) > 0 {
		patterns = allowPaths[0]
	}
	return &CreatePDFTool{
		workspace:   workspace,
		restrict:    restrict,
		maxFileSize: maxFileSize,
		mediaStore:  store,
		allowPaths:  patterns,
	}
}

func (t *CreatePDFTool) Name() string { return "create_pdf" }

func (t *CreatePDFTool) Description() string {
	return "Create a simple PDF document from text and send it to the user on the current chat channel."
}

func (t *CreatePDFTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "Optional document title displayed at the top of the PDF.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Plain text content to include in the PDF.",
			},
			"filename": map[string]any{
				"type":        "string",
				"description": "Optional PDF filename. Defaults to document.pdf.",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Optional output path. Relative paths are resolved from the workspace.",
			},
		},
		"required": []string{"content"},
	}
}

func (t *CreatePDFTool) SetMediaStore(store media.MediaStore) {
	t.mediaStore = store
}

func (t *CreatePDFTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	content, _ := args["content"].(string)
	if strings.TrimSpace(content) == "" {
		return ErrorResult("content is required")
	}
	if t.workspace == "" {
		return ErrorResult("workspace is not configured")
	}
	if t.mediaStore == nil {
		return ErrorResult("media store not configured")
	}

	channel := ToolChannel(ctx)
	chatID := ToolChatID(ctx)
	if channel == "" || chatID == "" {
		return ErrorResult("no target channel/chat available")
	}

	title, _ := args["title"].(string)
	filename, _ := args["filename"].(string)
	filename = ensurePDFFilename(sanitizePDFFilename(filename))
	if filename == ".pdf" {
		filename = "document.pdf"
	}

	outPath, _ := args["path"].(string)
	if strings.TrimSpace(outPath) == "" {
		outPath = filepath.Join(defaultPDFSubdir, filename)
	}
	resolved, err := validatePathWithAllowPaths(outPath, t.workspace, t.restrict, t.allowPaths)
	if err != nil {
		return ErrorResult(fmt.Sprintf("invalid output path: %v", err))
	}
	if filepath.Ext(strings.ToLower(resolved)) != ".pdf" {
		resolved += ".pdf"
	}
	if err = os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return ErrorResult(fmt.Sprintf("failed to prepare output directory: %v", err))
	}

	data := buildSimplePDF(title, content)
	if len(data) > t.maxFileSize {
		return ErrorResult(fmt.Sprintf("PDF too large: %d bytes (max %d bytes)", len(data), t.maxFileSize))
	}
	if err = os.WriteFile(resolved, data, 0o600); err != nil {
		return ErrorResult(fmt.Sprintf("failed to write PDF: %v", err))
	}

	scope := fmt.Sprintf("tool:create_pdf:%s:%s", channel, chatID)
	ref, err := t.mediaStore.Store(resolved, media.MediaMeta{
		Filename:      filepath.Base(resolved),
		ContentType:   "application/pdf",
		Source:        "tool:create_pdf",
		CleanupPolicy: media.CleanupPolicyForgetOnly,
	}, scope)
	if err != nil {
		return ErrorResult(fmt.Sprintf("failed to register PDF: %v", err))
	}

	return MediaResult(fmt.Sprintf("PDF %q created and sent to user", filepath.Base(resolved)), []string{ref})
}

func buildSimplePDF(title, content string) []byte {
	lines := pdfLines(title, content)
	pages := paginatePDFLines(lines)

	var body bytes.Buffer
	offsets := []int{0}
	writeObj := func(id int, s string) {
		for len(offsets) <= id {
			offsets = append(offsets, 0)
		}
		offsets[id] = body.Len()
		fmt.Fprintf(&body, "%d 0 obj\n%s\nendobj\n", id, s)
	}

	writeObj(1, "<< /Type /Catalog /Pages 2 0 R >>")
	kids := make([]string, len(pages))
	for i := range pages {
		kids[i] = fmt.Sprintf("%d 0 R", 3+i*2)
	}
	writeObj(2, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)))

	nextObj := 3
	fontObj := 3 + len(pages)*2
	for _, page := range pages {
		pageObj := nextObj
		contentObj := nextObj + 1
		writeObj(pageObj, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Resources << /Font << /F1 %d 0 R >> >> /Contents %d 0 R >>", pdfPageWidth, pdfPageHeight, fontObj, contentObj))
		stream := pdfContentStream(page)
		writeObj(contentObj, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream))
		nextObj += 2
	}
	writeObj(fontObj, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>")

	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	out.Write(body.Bytes())
	xrefStart := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n", len(offsets))
	out.WriteString("0000000000 65535 f \n")
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", len("%PDF-1.4\n")+offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets), xrefStart)
	return out.Bytes()
}

func pdfLines(title, content string) []string {
	var lines []string
	if strings.TrimSpace(title) != "" {
		lines = append(lines, strings.TrimSpace(title), "")
	}
	for _, paragraph := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		wrapped := wrapPDFLine(paragraph, pdfMaxLineRunes)
		lines = append(lines, wrapped...)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func wrapPDFLine(line string, maxRunes int) []string {
	line = strings.TrimRight(line, " \t")
	if line == "" {
		return []string{""}
	}
	var result []string
	var current strings.Builder
	for _, word := range strings.Fields(line) {
		nextLen := utf8.RuneCountInString(current.String())
		if nextLen > 0 {
			nextLen++
		}
		nextLen += utf8.RuneCountInString(word)
		if nextLen > maxRunes && current.Len() > 0 {
			result = append(result, current.String())
			current.Reset()
		}
		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(word)
	}
	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}

func paginatePDFLines(lines []string) [][]string {
	maxLines := (pdfPageHeight - 2*pdfMargin) / pdfLineHeight
	if maxLines < 1 {
		maxLines = 1
	}
	var pages [][]string
	for len(lines) > 0 {
		n := maxLines
		if len(lines) < n {
			n = len(lines)
		}
		page := append([]string(nil), lines[:n]...)
		pages = append(pages, page)
		lines = lines[n:]
	}
	if len(pages) == 0 {
		pages = append(pages, []string{""})
	}
	return pages
}

func pdfContentStream(lines []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "BT\n/F1 %d Tf\n%d TL\n%d %d Td\n", pdfFontSize, pdfLineHeight, pdfMargin, pdfPageHeight-pdfMargin)
	for i, line := range lines {
		if i > 0 {
			b.WriteString("T*\n")
		}
		b.WriteString("(")
		b.WriteString(escapePDFText(line))
		b.WriteString(") Tj\n")
	}
	b.WriteString("ET")
	return b.String()
}

func escapePDFText(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '(', ')':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\t':
			b.WriteByte(' ')
		default:
			if r >= 32 && r <= 126 {
				b.WriteRune(r)
			} else {
				b.WriteString("?")
			}
		}
	}
	return b.String()
}

func sanitizePDFFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return "document.pdf"
	}
	filename = filepath.Base(filename)
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "\x00", "")
	filename = replacer.Replace(filename)
	return strings.TrimSpace(filename)
}

func ensurePDFFilename(filename string) string {
	if filename == "" {
		filename = "document"
	}
	if strings.EqualFold(filepath.Ext(filename), ".pdf") {
		return filename
	}
	return filename + ".pdf"
}
