package elk

//IDQuery .
type IDQuery struct {
	Query struct {
		IDS struct {
			Type   string   `json:"type"`
			Values []string `json:"values"`
		} `json:"ids"`
	} `json:"query"`
}

//AllQuery .
type AllQuery struct {
	Query struct {
		MatchAll map[string]interface{} `json:"match_all"`
	} `json:"query"`
	Size int `json:"size"`
}

//MatchQuery .
type MatchQuery struct {
	Query struct {
		Match map[string]interface{} `json:"match"`
	} `json:"query"`
	Size int `json:"size"`
}
