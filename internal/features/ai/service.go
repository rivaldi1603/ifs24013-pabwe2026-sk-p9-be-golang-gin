package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Service defines AI Blog Generator business logic with live Search Engine integration.
type Service interface {
	GeneratePost(ctx context.Context, req GeneratePostRequest) (*GeneratePostResponse, error)
}

type service struct {
	aiClient     Client
	searchClient SearchClient
}

// NewService creates a new AI blog generation Service.
func NewService(aiClient Client, searchClient SearchClient) Service {
	return &service{
		aiClient:     aiClient,
		searchClient: searchClient,
	}
}

func (s *service) GeneratePost(ctx context.Context, req GeneratePostRequest) (*GeneratePostResponse, error) {
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Umum"
	}

	sources := []SearchSource{}
	var searchContext string
	if req.IncludeWebSearch && s.searchClient != nil {
		results, err := s.searchClient.Search(ctx, req.Topic)
		if err == nil && len(results) > 0 {
			sources = results
			var sb strings.Builder
			sb.WriteString("\nReferensi hasil penelusuran mesin pencari terkini:\n")
			for _, r := range results {
				sb.WriteString(fmt.Sprintf("- %s (%s): %s\n", r.Title, r.URL, r.Snippet))
			}
			searchContext = sb.String()
		}
	}

	systemPrompt := "You are an expert Indonesian blog writer. Return ONLY valid JSON with keys \"title\", \"category\", \"excerpt\", and \"content\" (rich Markdown article with citations when references are provided)."
	userPrompt := fmt.Sprintf(
		"Tuliskan draf artikel blog lengkap berbahasa Indonesia tentang topik \"%s\" untuk kategori \"%s\".%s",
		strings.TrimSpace(req.Topic),
		category,
		searchContext,
	)

	raw, err := s.aiClient.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	draft, err := parseGeneratedPost(raw)
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(draft.Category) == "" {
		draft.Category = category
	}
	draft.SearchSources = sources

	return draft, nil
}

func parseGeneratedPost(raw string) (*GeneratePostResponse, error) {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var res GeneratePostResponse
	if err := json.Unmarshal([]byte(cleaned), &res); err != nil {
		return nil, fmt.Errorf("failed to parse AI article JSON: %w", err)
	}
	if strings.TrimSpace(res.Title) == "" || strings.TrimSpace(res.Content) == "" {
		return nil, fmt.Errorf("AI response missing required title or content")
	}
	return &res, nil
}
