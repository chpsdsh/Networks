package domain

type WikiGeoSearchResp struct {
	Query struct {
		GeoSearch []GeoSearch `json:"geosearch"`
	} `json:"query"`
}

type GeoSearch struct {
	PageID int     `json:"pageid"`
	Title  string  `json:"title"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Dist   float64 `json:"dist"`
}

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

type WikiGeoSearchAndPlaceInfo struct {
	GeoSearch GeoSearch
	PlaceInfo PlaceInfo
}
