package server

// WebSearchParams represents the parameters for web_search tool
type WebSearchParams struct {
	Query           string   `json:"query" jsonschema:"The search query"`
	Engine          string   `json:"engine,omitempty" jsonschema:"Single search engine to use (mutually exclusive with engines)"`
	Engines         []string `json:"engines,omitempty" jsonschema:"List of search engines to use (mutually exclusive with engine)"`
	MaxResults      int      `json:"max_results,omitempty" jsonschema:"Maximum number of results to return (default: 10)"`
	Language        string   `json:"language,omitempty" jsonschema:"Language code for search results (e.g., 'en', 'zh')"`
	ArxivCategory   string   `json:"arxiv_category,omitempty" jsonschema:"Arxiv category for academic paper search (e.g., 'cs.AI', 'math.CO')"`
	SearchDepth     string   `json:"search_depth,omitempty" jsonschema:"Search depth: 'quick' (1 query), 'normal' (2 queries), 'deep' (3 queries). Default: 'normal'"`
	IncludeAcademic bool     `json:"include_academic,omitempty" jsonschema:"Include academic papers from Arxiv (default: false)"`
	AutoQueryExpand bool     `json:"auto_query_expand,omitempty" jsonschema:"Automatically expand query with variations (news, academic) based on search_depth (default: true)"`
	AutoDeduplicate bool     `json:"auto_deduplicate,omitempty" jsonschema:"Automatically deduplicate results by URL (default: true)"`
}

// WebFetchParams represents the parameters for web_fetch tool
type WebFetchParams struct {
	URL        string `json:"url" jsonschema:"URL of the website to fetch"`
	MaxLength  int    `json:"max_length,omitempty" jsonschema:"Maximum number of characters to return (default: 5000)"`
	StartIndex int    `json:"start_index,omitempty" jsonschema:"Start content from this character index (default: 0)"`
	Raw        bool   `json:"raw,omitempty" jsonschema:"If true, returns the raw HTML including <script> and <style> blocks (default: false)"`
}

// webSearchOutput defines the output structure for web_search tool
type webSearchOutput struct {
	Text string `json:"text"`
}

// webFetchOutput defines the output structure for web_fetch tool
type webFetchOutput struct {
	Text string `json:"text"`
}
