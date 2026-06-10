// JameClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 JameClaw contributors

package agent

import (
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/sipeed/jameclaw/pkg/fileutil"
)

// MemoryStore manages persistent memory for the agent.
// - Long-term memory: memory/MEMORY.md
// - Daily notes: memory/YYYYMM/YYYYMMDD.md
type MemoryStore struct {
	workspace  string
	memoryDir  string
	memoryFile string
}

type MemorySearchResult struct {
	Path    string
	Snippet string
	Score   float64
}

type memoryChunk struct {
	path string
	text string
}

const (
	memoryChunkRunes   = 900
	memoryChunkOverlap = 120
	memoryVectorSize   = 256
)

// NewMemoryStore creates a new MemoryStore with the given workspace path.
// It ensures the memory directory exists.
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")
	memoryFile := filepath.Join(memoryDir, "MEMORY.md")

	// Ensure memory directory exists
	os.MkdirAll(memoryDir, 0o755)

	return &MemoryStore{
		workspace:  workspace,
		memoryDir:  memoryDir,
		memoryFile: memoryFile,
	}
}

// getTodayFile returns the path to today's daily note file (memory/YYYYMM/YYYYMMDD.md).
func (ms *MemoryStore) getTodayFile() string {
	today := time.Now().Format("20060102") // YYYYMMDD
	monthDir := today[:6]                  // YYYYMM
	filePath := filepath.Join(ms.memoryDir, monthDir, today+".md")
	return filePath
}

// ReadLongTerm reads the long-term memory (MEMORY.md).
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadLongTerm() string {
	if data, err := os.ReadFile(ms.memoryFile); err == nil {
		return string(data)
	}
	return ""
}

// WriteLongTerm writes content to the long-term memory file (MEMORY.md).
func (ms *MemoryStore) WriteLongTerm(content string) error {
	// Use unified atomic write utility with explicit sync for flash storage reliability.
	// Using 0o600 (owner read/write only) for secure default permissions.
	return fileutil.WriteFileAtomic(ms.memoryFile, []byte(content), 0o600)
}

// ReadToday reads today's daily note.
// Returns empty string if the file doesn't exist.
func (ms *MemoryStore) ReadToday() string {
	todayFile := ms.getTodayFile()
	if data, err := os.ReadFile(todayFile); err == nil {
		return string(data)
	}
	return ""
}

// AppendToday appends content to today's daily note.
// If the file doesn't exist, it creates a new file with a date header.
func (ms *MemoryStore) AppendToday(content string) error {
	todayFile := ms.getTodayFile()

	// Ensure month directory exists
	monthDir := filepath.Dir(todayFile)
	if err := os.MkdirAll(monthDir, 0o755); err != nil {
		return err
	}

	var existingContent string
	if data, err := os.ReadFile(todayFile); err == nil {
		existingContent = string(data)
	}

	var newContent string
	if existingContent == "" {
		// Add header for new day
		header := fmt.Sprintf("# %s\n\n", time.Now().Format("2006-01-02"))
		newContent = header + content
	} else {
		// Append to existing content
		newContent = existingContent + "\n" + content
	}

	// Use unified atomic write utility with explicit sync for flash storage reliability.
	return fileutil.WriteFileAtomic(todayFile, []byte(newContent), 0o600)
}

// GetRecentDailyNotes returns daily notes from the last N days.
// Contents are joined with "---" separator.
func (ms *MemoryStore) GetRecentDailyNotes(days int) string {
	var sb strings.Builder
	first := true

	for i := range days {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("20060102") // YYYYMMDD
		monthDir := dateStr[:6]            // YYYYMM
		filePath := filepath.Join(ms.memoryDir, monthDir, dateStr+".md")

		if data, err := os.ReadFile(filePath); err == nil {
			if !first {
				sb.WriteString("\n\n---\n\n")
			}
			sb.Write(data)
			first = false
		}
	}

	return sb.String()
}

// GetMemoryContext returns formatted memory context for the agent prompt.
// Includes long-term memory and recent daily notes.
func (ms *MemoryStore) GetMemoryContext() string {
	longTerm := ms.ReadLongTerm()
	recentNotes := ms.GetRecentDailyNotes(3)

	if longTerm == "" && recentNotes == "" {
		return ""
	}

	var sb strings.Builder

	if longTerm != "" {
		sb.WriteString("## Long-term Memory\n\n")
		sb.WriteString(longTerm)
	}

	if recentNotes != "" {
		if longTerm != "" {
			sb.WriteString("\n\n---\n\n")
		}
		sb.WriteString("## Recent Daily Notes\n\n")
		sb.WriteString(recentNotes)
	}

	return sb.String()
}

// Search returns only the memory snippets most relevant to query. Ranking is
// hybrid: BM25-style lexical relevance is combined with a local hashed concept
// vector, which catches common preference wording variations without an API call.
func (ms *MemoryStore) Search(query string, limit, maxChars int) []MemorySearchResult {
	query = strings.TrimSpace(query)
	if query == "" || limit <= 0 || maxChars <= 0 {
		return nil
	}

	chunks := ms.loadSearchChunks()
	if len(chunks) == 0 {
		return nil
	}

	queryTerms := memoryTerms(query)
	if len(queryTerms) == 0 {
		return nil
	}

	docTerms := make([][]string, len(chunks))
	docFreq := make(map[string]int)
	totalTerms := 0
	for i, chunk := range chunks {
		docTerms[i] = memoryTerms(chunk.text)
		totalTerms += len(docTerms[i])
		seen := make(map[string]struct{})
		for _, term := range docTerms[i] {
			if _, ok := seen[term]; ok {
				continue
			}
			seen[term] = struct{}{}
			docFreq[term]++
		}
	}
	avgLen := float64(totalTerms) / float64(len(chunks))
	queryVector := memoryVector(query)

	results := make([]MemorySearchResult, 0, len(chunks))
	for i, chunk := range chunks {
		lexical := bm25Score(queryTerms, docTerms[i], docFreq, len(chunks), avgLen)
		semantic := cosineSimilarity(queryVector, memoryVector(chunk.text))
		score := lexical + semantic*1.5
		if score <= 0.05 {
			continue
		}
		results = append(results, MemorySearchResult{Path: chunk.path, Snippet: chunk.text, Score: score})
	}

	sort.SliceStable(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > limit {
		results = results[:limit]
	}

	used := 0
	trimmed := results[:0]
	for _, result := range results {
		remaining := maxChars - used
		if remaining <= 0 {
			break
		}
		runes := []rune(result.Snippet)
		if len(runes) > remaining {
			result.Snippet = strings.TrimSpace(string(runes[:remaining]))
		}
		if result.Snippet == "" {
			continue
		}
		used += len([]rune(result.Snippet))
		trimmed = append(trimmed, result)
	}
	return trimmed
}

func (ms *MemoryStore) loadSearchChunks() []memoryChunk {
	var chunks []memoryChunk
	_ = filepath.WalkDir(ms.memoryDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(ms.workspace, path)
		if relErr != nil {
			rel = path
		}
		chunks = append(chunks, splitMemoryChunks(filepath.ToSlash(rel), string(data))...)
		return nil
	})
	return chunks
}

func splitMemoryChunks(path, content string) []memoryChunk {
	var chunks []memoryChunk
	var heading string
	for _, block := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n\n") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		if strings.HasPrefix(block, "#") && !strings.Contains(block, "\n") {
			heading = block
			continue
		}
		if heading != "" {
			block = heading + "\n\n" + block
		}
		chunks = append(chunks, splitLongMemoryBlock(path, block)...)
	}
	if len(chunks) == 0 && heading != "" {
		chunks = append(chunks, memoryChunk{path: path, text: heading})
	}
	return chunks
}

func splitLongMemoryBlock(path, content string) []memoryChunk {
	runes := []rune(content)
	var chunks []memoryChunk
	for start := 0; start < len(runes); {
		end := min(start+memoryChunkRunes, len(runes))
		chunks = append(chunks, memoryChunk{path: path, text: strings.TrimSpace(string(runes[start:end]))})
		if end == len(runes) {
			break
		}
		start = end - memoryChunkOverlap
	}
	return chunks
}

func memoryTerms(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func bm25Score(query, document []string, docFreq map[string]int, documentCount int, avgLen float64) float64 {
	if len(document) == 0 || avgLen == 0 {
		return 0
	}
	counts := make(map[string]int)
	for _, term := range document {
		counts[term]++
	}
	const k1, b = 1.2, 0.75
	score := 0.0
	seen := make(map[string]struct{})
	for _, term := range query {
		if _, ok := seen[term]; ok {
			continue
		}
		seen[term] = struct{}{}
		tf := float64(counts[term])
		if tf == 0 {
			continue
		}
		idf := math.Log(1 + (float64(documentCount-docFreq[term])+0.5)/(float64(docFreq[term])+0.5))
		denominator := tf + k1*(1-b+b*float64(len(document))/avgLen)
		score += idf * tf * (k1 + 1) / denominator
	}
	return score
}

func memoryVector(text string) []float64 {
	vector := make([]float64, memoryVectorSize)
	terms := semanticMemoryTerms(text)
	for i, term := range terms {
		addMemoryFeature(vector, term, 1)
		if i > 0 {
			addMemoryFeature(vector, terms[i-1]+" "+term, 0.5)
		}
	}
	return vector
}

func semanticMemoryTerms(text string) []string {
	raw := memoryTerms(text)
	terms := make([]string, 0, len(raw))
	for _, term := range raw {
		if len(term) < 3 || memoryStopWords[term] {
			continue
		}
		if concept, ok := memoryConcepts[term]; ok {
			term = concept
		} else {
			term = stemMemoryTerm(term)
		}
		terms = append(terms, term)
	}
	return terms
}

func addMemoryFeature(vector []float64, feature string, weight float64) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(feature))
	vector[int(h.Sum32()%uint32(len(vector)))] += weight
}

func stemMemoryTerm(term string) string {
	for _, suffix := range []string{"ingly", "edly", "ation", "ments", "ment", "ness", "ing", "ers", "ies", "ed", "es", "s"} {
		if len(term) > len(suffix)+3 && strings.HasSuffix(term, suffix) {
			return strings.TrimSuffix(term, suffix)
		}
	}
	return term
}

var memoryConcepts = map[string]string{
	"adore":       "prefer",
	"enjoy":       "prefer",
	"favorite":    "prefer",
	"favourite":   "prefer",
	"like":        "prefer",
	"likes":       "prefer",
	"love":        "prefer",
	"loves":       "prefer",
	"preference":  "prefer",
	"preferences": "prefer",
	"preferred":   "prefer",
	"prefers":     "prefer",
	"dislike":     "avoid",
	"dislikes":    "avoid",
	"hate":        "avoid",
	"hates":       "avoid",
	"never":       "avoid",
	"timezone":    "location",
	"city":        "location",
	"town":        "location",
	"job":         "work",
	"profession":  "work",
	"role":        "work",
}

var memoryStopWords = map[string]bool{
	"about": true, "and": true, "are": true, "for": true, "from": true,
	"has": true, "have": true, "how": true, "the": true, "their": true,
	"them": true, "they": true, "this": true, "user": true, "what": true,
	"when": true, "where": true, "which": true, "with": true, "would": true,
}

func cosineSimilarity(a, b []float64) float64 {
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / math.Sqrt(normA*normB)
}
