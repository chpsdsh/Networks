package domain

type WikiPlaceInfo struct {
	Query struct {
		Pages []PlaceInfo `json:"pages"`
	} `json:"query"`
}

type PlaceInfo struct {
	Title   string `json:"title"`
	Extract string `json:"extract"`
	FullURL string `json:"fullurl"`
}
