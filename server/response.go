package server

type WebSearchResponse struct {
	TotalResults int            `json:"total_results"`
	Summary      SearchSummary  `json:"summary"`
	Results      []SearchResult `json:"results"`
	SearchTime   string         `json:"search_time"`
}

type SearchSummary struct {
	TotalRawResults    int      `json:"total_raw_results"`
	TotalUniqueResults int      `json:"total_unique_results"`
	OriginalQuery      string   `json:"original_query"`
	SearchQueries      []string `json:"search_queries"`
	EnginesUsed        []string `json:"engines_used"`
	SearchDepth        string   `json:"search_depth"`
}

type SearchResult struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}
