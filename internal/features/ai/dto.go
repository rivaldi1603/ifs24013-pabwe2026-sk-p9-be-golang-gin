package ai

// GeneratePostRequest represents the payload for AI Blog Generator with Search Engine integration.
type GeneratePostRequest struct {
	Topic            string `json:"topic" binding:"required"`
	Category         string `json:"category"`
	IncludeWebSearch bool   `json:"include_web_search"`
}

// SearchSource represents a web search reference returned alongside the generated article.
type SearchSource struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// GeneratePostResponse represents a complete AI-generated article draft and its search sources.
type GeneratePostResponse struct {
	Title         string         `json:"title"`
	Category      string         `json:"category"`
	Excerpt       string         `json:"excerpt"`
	Content       string         `json:"content"`
	SearchSources []SearchSource `json:"search_sources"`
}
