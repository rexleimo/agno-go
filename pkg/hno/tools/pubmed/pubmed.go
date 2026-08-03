package pubmed

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

// PubMedToolkit provides access to PubMed biomedical literature
// This is a simplified implementation that provides basic PubMed search capabilities

type PubMedToolkit struct {
	*toolkit.BaseToolkit
}

// New creates a new PubMed toolkit
func New() *PubMedToolkit {
	t := &PubMedToolkit{
		BaseToolkit: toolkit.NewBaseToolkit("pubmed"),
	}

	// Register PubMed search function
	t.RegisterFunction(&toolkit.Function{
		Name:        "search_articles",
		Description: "Search for biomedical articles on PubMed",
		Parameters: map[string]toolkit.Parameter{
			"query": {
				Type:        "string",
				Description: "Search query for articles",
				Required:    true,
			},
			"max_results": {
				Type:        "integer",
				Description: "Maximum number of results to return (default: 10)",
				Required:    false,
				Default:     10,
			},
		},
		Handler: t.searchArticles,
	})

	// Register PubMed article details function
	t.RegisterFunction(&toolkit.Function{
		Name:        "get_article_details",
		Description: "Get detailed information about a specific PubMed article",
		Parameters: map[string]toolkit.Parameter{
			"article_id": {
				Type:        "string",
				Description: "PubMed article ID (PMID)",
				Required:    true,
			},
		},
		Handler: t.getArticleDetails,
	})

	return t
}

// PubMed API response structures
type PubMedResponse struct {
	XMLName xml.Name `xml:"eSearchResult"`
	Count   string   `xml:"Count"`
	IdList  IdList   `xml:"IdList"`
}

type IdList struct {
	IDs []string `xml:"Id"`
}

type PubMedSummaryResponse struct {
	XMLName xml.Name `xml:"eSummaryResult"`
	Docs    []Doc    `xml:"DocSum"`
}

type Doc struct {
	ID    string   `xml:"Id,attr"`
	Items []Item   `xml:"Item"`
}

type Item struct {
	Name  string `xml:"Name,attr"`
	Type  string `xml:"Type,attr"`
	Value string `xml:",chardata"`
}

// searchArticles searches for articles on PubMed
func (p *PubMedToolkit) searchArticles(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	query, ok := args["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query must be a string")
	}

	maxResults := 10
	if maxArg, ok := args["max_results"].(float64); ok {
		maxResults = int(maxArg)
	}

	// Build PubMed ESearch API URL
	baseURL := "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi"
	params := url.Values{}
	params.Add("db", "pubmed")
	params.Add("term", query)
	params.Add("retmax", fmt.Sprintf("%d", maxResults))
	params.Add("retmode", "xml")
	params.Add("sort", "relevance")

	apiURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Make HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from PubMed API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PubMed API returned status %d", resp.StatusCode)
	}

	// Parse XML response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var pubmedResponse PubMedResponse
	if err := xml.Unmarshal(body, &pubmedResponse); err != nil {
		return nil, fmt.Errorf("failed to parse PubMed response: %w", err)
	}

	// Get article details for the IDs
	if len(pubmedResponse.IdList.IDs) == 0 {
		return map[string]interface{}{
			"query":       query,
			"results":     []interface{}{},
			"total_found": 0,
			"max_results": maxResults,
		}, nil
	}

	// Get summaries for the article IDs
	summaries, err := p.getArticleSummaries(pubmedResponse.IdList.IDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get article summaries: %w", err)
	}

	return map[string]interface{}{
		"query":       query,
		"results":     summaries,
		"total_found": len(pubmedResponse.IdList.IDs),
		"max_results": maxResults,
	}, nil
}

// getArticleDetails gets detailed information about a specific article
func (p *PubMedToolkit) getArticleDetails(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	articleID, ok := args["article_id"].(string)
	if !ok {
		return nil, fmt.Errorf("article_id must be a string")
	}

	// Get summary for the specific article
	summaries, err := p.getArticleSummaries([]string{articleID})
	if err != nil {
		return nil, err
	}

	if len(summaries) == 0 {
		return nil, fmt.Errorf("article with ID '%s' not found", articleID)
	}

	return map[string]interface{}{
		"article": summaries[0],
	}, nil
}

// getArticleSummaries gets summaries for a list of article IDs
func (p *PubMedToolkit) getArticleSummaries(articleIDs []string) ([]map[string]interface{}, error) {
	if len(articleIDs) == 0 {
		return []map[string]interface{}{}, nil
	}

	// Build PubMed ESummary API URL
	baseURL := "https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi"
	params := url.Values{}
	params.Add("db", "pubmed")
	params.Add("id", strings.Join(articleIDs, ","))
	params.Add("retmode", "xml")

	apiURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Make HTTP request
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from PubMed API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PubMed API returned status %d", resp.StatusCode)
	}

	// Parse XML response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var summaryResponse PubMedSummaryResponse
	if err := xml.Unmarshal(body, &summaryResponse); err != nil {
		return nil, fmt.Errorf("failed to parse PubMed summary response: %w", err)
	}

	// Convert to structured results
	results := make([]map[string]interface{}, 0)
	for _, doc := range summaryResponse.Docs {
		article := map[string]interface{}{
			"article_id": doc.ID,
		}

		// Extract information from items
		for _, item := range doc.Items {
			switch item.Name {
			case "Title":
				article["title"] = strings.TrimSpace(item.Value)
			case "FullJournalName":
				article["journal"] = strings.TrimSpace(item.Value)
			case "PubDate":
				article["publication_date"] = strings.TrimSpace(item.Value)
			case "Author":
				if authors, ok := article["authors"].([]string); ok {
					article["authors"] = append(authors, strings.TrimSpace(item.Value))
				} else {
					article["authors"] = []string{strings.TrimSpace(item.Value)}
				}
			case "DOI":
				article["doi"] = strings.TrimSpace(item.Value)
			case "Abstract":
				article["abstract"] = strings.TrimSpace(item.Value)
			}
		}

		// Ensure authors is always an array
		if _, ok := article["authors"]; !ok {
			article["authors"] = []string{}
		}

		// Build URL
		article["url"] = fmt.Sprintf("https://pubmed.ncbi.nlm.nih.gov/%s/", doc.ID)

		results = append(results, article)
	}

	return results, nil
}