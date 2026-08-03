package knowledge

import (
	"fmt"
	"strings"
	"unicode"
)

// Chunker interface for splitting documents into chunks
type Chunker interface {
	Chunk(doc Document) ([]Chunk, error)
}

// CharacterChunker splits documents by character count
type CharacterChunker struct {
	ChunkSize    int // Number of characters per chunk
	ChunkOverlap int // Number of characters to overlap between chunks
	Separator    string
}

// NewCharacterChunker creates a new character-based chunker
func NewCharacterChunker(chunkSize, chunkOverlap int) *CharacterChunker {
	if chunkSize <= 0 {
		chunkSize = 1000
	}
	if chunkOverlap < 0 || chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 10 // 10% overlap by default
	}

	return &CharacterChunker{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
		Separator:    "\n\n",
	}
}

// Chunk splits a document into character-based chunks
func (c *CharacterChunker) Chunk(doc Document) ([]Chunk, error) {
	text := doc.Content
	if len(text) == 0 {
		return []Chunk{}, nil
	}

	var chunks []Chunk
	index := 0
	start := 0

	for start < len(text) {
		end := start + c.ChunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at a separator or word boundary
		if end < len(text) {
			// Look for separator within the last 20% of the chunk
			searchStart := start + (c.ChunkSize * 4 / 5)
			if searchStart < start {
				searchStart = start
			}

			separatorIdx := strings.LastIndex(text[searchStart:end], c.Separator)
			if separatorIdx != -1 {
				end = searchStart + separatorIdx + len(c.Separator)
			} else {
				// No separator found, try to break at word boundary
				for end < len(text) && !unicode.IsSpace(rune(text[end])) {
					end++
				}
			}
		}

		content := strings.TrimSpace(text[start:end])
		if content != "" {
			chunk := Chunk{
				ID:      fmt.Sprintf("%s_chunk_%d", doc.ID, index),
				Content: content,
				Index:   index,
				Metadata: map[string]interface{}{
					"document_id": doc.ID,
					"source":      doc.Source,
					"chunk_index": index,
					"start_char":  start,
					"end_char":    end,
				},
			}

			// Copy document metadata
			if doc.Metadata != nil {
				for k, v := range doc.Metadata {
					if _, exists := chunk.Metadata[k]; !exists {
						chunk.Metadata[k] = v
					}
				}
			}

			chunks = append(chunks, chunk)
			index++
		}

		if end == len(text) {
			break
		}

		// Move start position with overlap
		start = end - c.ChunkOverlap
		if start < 0 {
			start = 0
		}
	}

	return chunks, nil
}

// SentenceChunker splits documents by sentences
type SentenceChunker struct {
	MaxChunkSize int // Maximum characters per chunk
	MinChunkSize int // Minimum characters per chunk
}

// NewSentenceChunker creates a new sentence-based chunker
func NewSentenceChunker(maxChunkSize, minChunkSize int) *SentenceChunker {
	if maxChunkSize <= 0 {
		maxChunkSize = 1000
	}
	if minChunkSize <= 0 || minChunkSize >= maxChunkSize {
		minChunkSize = maxChunkSize / 4
	}

	return &SentenceChunker{
		MaxChunkSize: maxChunkSize,
		MinChunkSize: minChunkSize,
	}
}

// Chunk splits a document into sentence-based chunks
func (c *SentenceChunker) Chunk(doc Document) ([]Chunk, error) {
	sentences := c.splitSentences(doc.Content)
	if len(sentences) == 0 {
		return []Chunk{}, nil
	}

	var chunks []Chunk
	var currentChunk strings.Builder
	index := 0

	for i, sentence := range sentences {
		sentenceLen := len(sentence)

		// If adding this sentence exceeds max size, create a new chunk
		if currentChunk.Len() > 0 && currentChunk.Len()+sentenceLen > c.MaxChunkSize {
			// Save current chunk
			chunks = append(chunks, c.createChunk(doc, currentChunk.String(), index))
			index++
			currentChunk.Reset()
		}

		// Add sentence to current chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString(" ")
		}
		currentChunk.WriteString(sentence)

		// If this is the last sentence or current chunk is >= min size
		isLast := i == len(sentences)-1
		if isLast || currentChunk.Len() >= c.MinChunkSize {
			if isLast || currentChunk.Len() >= c.MaxChunkSize {
				chunks = append(chunks, c.createChunk(doc, currentChunk.String(), index))
				index++
				currentChunk.Reset()
			}
		}
	}

	// Add any remaining content
	if currentChunk.Len() > 0 {
		chunks = append(chunks, c.createChunk(doc, currentChunk.String(), index))
	}

	return chunks, nil
}

func (c *SentenceChunker) createChunk(doc Document, content string, index int) Chunk {
	chunk := Chunk{
		ID:      fmt.Sprintf("%s_chunk_%d", doc.ID, index),
		Content: strings.TrimSpace(content),
		Index:   index,
		Metadata: map[string]interface{}{
			"document_id": doc.ID,
			"source":      doc.Source,
			"chunk_index": index,
		},
	}

	// Copy document metadata
	if doc.Metadata != nil {
		for k, v := range doc.Metadata {
			if _, exists := chunk.Metadata[k]; !exists {
				chunk.Metadata[k] = v
			}
		}
	}

	return chunk
}

// splitSentences splits text into sentences
func (c *SentenceChunker) splitSentences(text string) []string {
	// Simple sentence splitting on common sentence terminators
	var sentences []string
	var currentSentence strings.Builder

	for i, char := range text {
		currentSentence.WriteRune(char)

		// Check for sentence terminators
		if char == '.' || char == '!' || char == '?' {
			// Look ahead to see if this is really end of sentence
			if i+1 < len(text) {
				nextChar := rune(text[i+1])
				// If next char is space or newline, it's likely end of sentence
				if unicode.IsSpace(nextChar) {
					sentence := strings.TrimSpace(currentSentence.String())
					if len(sentence) > 0 {
						sentences = append(sentences, sentence)
					}
					currentSentence.Reset()
				}
			} else {
				// Last character
				sentence := strings.TrimSpace(currentSentence.String())
				if len(sentence) > 0 {
					sentences = append(sentences, sentence)
				}
			}
		}
	}

	// Add any remaining content
	if currentSentence.Len() > 0 {
		sentence := strings.TrimSpace(currentSentence.String())
		if len(sentence) > 0 {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// ParagraphChunker splits documents by paragraphs
type ParagraphChunker struct {
	MaxChunkSize int // Maximum characters per chunk
}

// NewParagraphChunker creates a new paragraph-based chunker
func NewParagraphChunker(maxChunkSize int) *ParagraphChunker {
	if maxChunkSize <= 0 {
		maxChunkSize = 2000
	}

	return &ParagraphChunker{
		MaxChunkSize: maxChunkSize,
	}
}

// Chunk splits a document into paragraph-based chunks
func (c *ParagraphChunker) Chunk(doc Document) ([]Chunk, error) {
	// Split by double newlines (common paragraph separator)
	paragraphs := strings.Split(doc.Content, "\n\n")

	var chunks []Chunk
	var currentChunk strings.Builder
	index := 0

	for i, para := range paragraphs {
		para = strings.TrimSpace(para)
		if len(para) == 0 {
			continue
		}

		paraLen := len(para)

		// If this paragraph alone exceeds max size, split it further
		if paraLen > c.MaxChunkSize {
			// Save current chunk if any
			if currentChunk.Len() > 0 {
				chunks = append(chunks, c.createChunk(doc, currentChunk.String(), index))
				index++
				currentChunk.Reset()
			}

			// Split large paragraph using character chunker
			tempDoc := Document{
				ID:       doc.ID,
				Content:  para,
				Source:   doc.Source,
				Metadata: doc.Metadata,
			}
			charChunker := NewCharacterChunker(c.MaxChunkSize, 100)
			paraChunks, _ := charChunker.Chunk(tempDoc)

			for _, chunk := range paraChunks {
				chunk.Index = index
				chunks = append(chunks, chunk)
				index++
			}
			continue
		}

		// If adding this paragraph exceeds max size, create a new chunk
		if currentChunk.Len() > 0 && currentChunk.Len()+paraLen+2 > c.MaxChunkSize {
			chunks = append(chunks, c.createChunk(doc, currentChunk.String(), index))
			index++
			currentChunk.Reset()
		}

		// Add paragraph to current chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(para)

		// If this is the last paragraph, save the chunk
		if i == len(paragraphs)-1 {
			chunks = append(chunks, c.createChunk(doc, currentChunk.String(), index))
		}
	}

	// Add any remaining content
	if currentChunk.Len() > 0 {
		chunks = append(chunks, c.createChunk(doc, currentChunk.String(), index))
	}

	return chunks, nil
}

func (c *ParagraphChunker) createChunk(doc Document, content string, index int) Chunk {
	chunk := Chunk{
		ID:      fmt.Sprintf("%s_chunk_%d", doc.ID, index),
		Content: strings.TrimSpace(content),
		Index:   index,
		Metadata: map[string]interface{}{
			"document_id": doc.ID,
			"source":      doc.Source,
			"chunk_index": index,
		},
	}

	// Copy document metadata
	if doc.Metadata != nil {
		for k, v := range doc.Metadata {
			if _, exists := chunk.Metadata[k]; !exists {
				chunk.Metadata[k] = v
			}
		}
	}

	return chunk
}
