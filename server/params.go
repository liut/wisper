package server

// WebSearchParams represents the parameters for web_search tool
type WebSearchParams struct {
	Query         string `json:"query"`
	Engine        string `json:"engine"`
	MaxResults    int    `json:"max_results"`
	Language      string `json:"language"`
	ArxivCategory string `json:"arxiv_category"`
}

// MultiSearchParams represents the parameters for multi_search tool
type MultiSearchParams struct {
	Query               string   `json:"query"`
	Engines             []string `json:"engines"`
	MaxResultsPerEngine int      `json:"max_results_per_engine"`
}

// SmartSearchParams represents the parameters for smart_search tool
type SmartSearchParams struct {
	Question        string `json:"question"`
	SearchDepth     string `json:"search_depth"`
	IncludeAcademic bool   `json:"include_academic"`
}

// WebFetchParams represents the parameters for web_fetch tool
type WebFetchParams struct {
	URL       string `json:"url"`
	MaxLength int    `json:"max_length"`
	StartIndex int   `json:"start_index"`
	Raw       bool   `json:"raw"`
}
