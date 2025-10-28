package domain

type WikiGeosearchResp struct {
	Query struct {
		Geosearch []struct {
			PageID int     `json:"pageid"`
			Title  string  `json:"title"`
			Lat    float64 `json:"lat"`
			Lon    float64 `json:"lon"`
			Dist   float64 `json:"dist"`
		} `json:"geosearch"`
	} `json:"query"`
}
