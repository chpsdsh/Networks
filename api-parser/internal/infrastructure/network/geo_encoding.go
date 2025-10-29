package network

import (
	"api-parser/internal/domain"
	"api-parser/internal/infrastructure/network/utils"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

func ReadGeoDataAsync(ctx context.Context, client *http.Client, query string) <-chan utils.Result[domain.GeoResponse] {
	out := make(chan utils.Result[domain.GeoResponse], 1)
	go func() {
		defer close(out)

		key := os.Getenv("GRAPHOPPER_KEY")
		if key == "" {
			select {
			case out <- utils.Result[domain.GeoResponse]{Err: fmt.Errorf("env var GRAPHOPPER_KEY not set")}:
			case <-ctx.Done():
			}
			return
		}

		u := fmt.Sprintf(
			"https://graphhopper.com/api/1/geocode?q=%s&key=%s",
			url.QueryEscape(query), key,
		)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			select {
			case out <- utils.Result[domain.GeoResponse]{Err: fmt.Errorf("error creating request: %w", err)}:
			case <-ctx.Done():
			}
			return
		}

		var geoResponse domain.GeoResponse
		if err := utils.DoJSON(client, req, &geoResponse); err != nil {
			select {
			case out <- utils.Result[domain.GeoResponse]{Err: fmt.Errorf("error doing request: %w", err)}:
			case <-ctx.Done():
			}
			return
		}

		select {
		case out <- utils.Result[domain.GeoResponse]{Value: geoResponse}:
		case <-ctx.Done():
			return
		}
	}()
	return out
}
