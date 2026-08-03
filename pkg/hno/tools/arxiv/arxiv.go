package arxiv

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/tools/toolkit"
)

// ArXivToolkit provides access to arXiv scientific papers
// This is a simplified implementation that provides basic arXiv search capabilities

const defaultAPIBase = "http://export.arxiv.org/api/query"

type ArXivToolkit struct {
	*toolkit.BaseToolkit
	// apiBase is the arXiv API endpoint; overridable for tests.
	// apiBase 是 arXiv API 端点；测试时可覆盖。
	apiBase string
}

// New creates a new ArXiv toolkit
func New() *ArXivToolkit {
	return NewWithBase(defaultAPIBase)
}

// NewWithBase creates a toolkit targeting a custom API endpoint (used by
// tests to avoid hitting the real arXiv service).
// NewWithBase 创建指向自定义 API 端点的 toolkit（测试用，避免打真实 arXiv）。
func NewWithBase(apiBase string) *ArXivToolkit {
	t := &ArXivToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("arxiv"),
		apiBase:     apiBase,
	}

	// Register arXiv search function
	t.RegisterFunction(&toolkit.Function{
		Name:        "search_papers",
		Description: "Search for scientific papers on arXiv",
		Parameters: map[string]toolkit.Parameter{
			"query": {
				Type:        "string",
				Description: "Search query for papers",
				Required:    true,
			},
			"max_results": {
				Type:        "integer",
				Description: "Maximum number of results to return (default: 10)",
				Required:    false,
				Default:     10,
			},
		},
		Handler: t.searchPapers,
	})

	// Register arXiv paper details function
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_paper_details",
		Description: "Get detailed information about a specific arXiv paper",
		Parameters: map[string]toolkit.Parameter{
			"paper_id": {
				Type:        "string",
				Description: "arXiv paper ID (e.g., '2301.00001')",
				Required:    true,
			},
		},
		Handler: t.getPaperDetails,
	})

	return t
}

// arXiv API response structures
type ArXivResponse struct {
	XMLName xml.Name `xml:"feed"`
	Entries []Entry  `xml:"entry"`
}

type Entry struct {
	ID        string   `xml:"id"`
	Title     string   `xml:"title"`
	Summary   string   `xml:"summary"`
	Published string   `xml:"published"`
	Updated   string   `xml:"updated"`
	Authors   []Author `xml:"author"`
	Links     []Link   `xml:"link"`
	Category  Category `xml:"category"`
}

type Author struct {
	Name string `xml:"name"`
}

type Link struct {
	Href  string `xml:"href,attr"`
	Rel   string `xml:"rel,attr"`
	Title string `xml:"title,attr"`
}

type Category struct {
	Term string `xml:"term,attr"`
}

// searchPapers searches for papers on arXiv
func (a *ArXivToolkit) searchPapers(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	maxResults := 10
	if maxArg, ok := args["max_results"].(float64); ok {
		maxResults = int(maxArg)
	}

	// Build arXiv API URL
	baseURL := a.apiBase
	params := url.Values{}
	params.Add("search_query", query)
	params.Add("start", "0")
	params.Add("max_results", fmt.Sprintf("%d", maxResults))
	params.Add("sortBy", "relevance")
	params.Add("sortOrder", "descending")

	apiURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Make HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from arXiv API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arXiv API returned status %d", resp.StatusCode)
	}

	// Parse XML response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var arxivResponse ArXivResponse
	if err := xml.Unmarshal(body, &arxivResponse); err != nil {
		return nil, fmt.Errorf("failed to parse arXiv response: %w", err)
	}

	// Convert to structured results
	results := make([]map[string]interface{}, 0)
	for _, entry := range arxivResponse.Entries {
		if len(results) >= maxResults {
			break
		}

		// Extract paper ID from URL
		paperID := ""
		if strings.Contains(entry.ID, "/") {
			parts := strings.Split(entry.ID, "/")
			if len(parts) > 0 {
				paperID = parts[len(parts)-1]
			}
		}

		// Extract author names
		authorNames := make([]string, len(entry.Authors))
		for i, author := range entry.Authors {
			authorNames[i] = author.Name
		}

		result := map[string]interface{}{
			"paper_id":  paperID,
			"title":     strings.TrimSpace(entry.Title),
			"summary":   strings.TrimSpace(entry.Summary),
			"authors":   authorNames,
			"published": entry.Published,
			"updated":   entry.Updated,
			"category":  entry.Category.Term,
			"url":       entry.ID,
		}

		results = append(results, result)
	}

	return map[string]interface{}{
		"query":       query,
		"results":     results,
		"total_found": len(results),
		"max_results": maxResults,
	}, nil
}

// getPaperDetails gets detailed information about a specific paper
func (a *ArXivToolkit) getPaperDetails(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	paperID, ok := args["paper_id"].(string)
	if !ok {
		return nil, fmt.Errorf("paper_id must be a string")
	}

	// Build arXiv API URL for specific paper
	baseURL := a.apiBase
	params := url.Values{}
	params.Add("id_list", paperID)

	apiURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Make HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from arXiv API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("arXiv API returned status %d", resp.StatusCode)
	}

	// Parse XML response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var arxivResponse ArXivResponse
	if err := xml.Unmarshal(body, &arxivResponse); err != nil {
		return nil, fmt.Errorf("failed to parse arXiv response: %w", err)
	}

	if len(arxivResponse.Entries) == 0 {
		return nil, fmt.Errorf("paper with ID '%s' not found", paperID)
	}

	entry := arxivResponse.Entries[0]

	// Extract author names
	authorNames := make([]string, len(entry.Authors))
	for i, author := range entry.Authors {
		authorNames[i] = author.Name
	}

	// Extract PDF link
	pdfURL := ""
	for _, link := range entry.Links {
		if link.Title == "pdf" {
			pdfURL = link.Href
			break
		}
	}

	result := map[string]interface{}{
		"paper_id":  paperID,
		"title":     strings.TrimSpace(entry.Title),
		"summary":   strings.TrimSpace(entry.Summary),
		"authors":   authorNames,
		"published": entry.Published,
		"updated":   entry.Updated,
		"category":  entry.Category.Term,
		"url":       entry.ID,
		"pdf_url":   pdfURL,
	}

	return map[string]interface{}{
		"paper": result,
	}, nil
}
