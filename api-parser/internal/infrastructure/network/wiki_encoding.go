package network

import (
	"api-parser/internal/domain"
	"api-parser/internal/infrastructure/network/utils"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

func ReadWikiDataAsync(ctx context.Context, client *http.Client, lat, lng float64) <-chan utils.Result[domain.WikiGeoSearchResp] {
	out := make(chan utils.Result[domain.WikiGeoSearchResp], 1)
	go func() {
		defer close(out)

		q := url.Values{}
		q.Set("action", "query")
		q.Set("list", "geosearch")
		q.Set("gscoord", fmt.Sprintf("%f|%f", lat, lng))
		q.Set("gsradius", strconv.Itoa(1000))
		q.Set("gslimit", strconv.Itoa(10))
		q.Set("format", "json")

		u := "https://en.wikipedia.org/w/api.php?" + q.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "api-parser/1.0")

		if err != nil {
			select {
			case out <- utils.Result[domain.WikiGeoSearchResp]{Err: err}:
			case <-ctx.Done():
			}
			return
		}

		var resp domain.WikiGeoSearchResp
		if err := utils.DoJSON(client, req, &resp); err != nil {
			select {
			case out <- utils.Result[domain.WikiGeoSearchResp]{Err: fmt.Errorf("wikipedia geosearch: %w", err)}:
			case <-ctx.Done():
			}
		}

		select {
		case out <- utils.Result[domain.WikiGeoSearchResp]{Value: resp}:
		case <-ctx.Done():
		}
	}()
	return out
}
