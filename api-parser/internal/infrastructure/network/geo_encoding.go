package network

import (
	"api-parser/internal/domain"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

func ReadGeoDataAsync(ctx context.Context, client *http.Client, query string) <-chan Result[domain.GeoResponse] {
	out := make(chan Result[domain.GeoResponse], 1)
	go func() {
		defer close(out)

		key := os.Getenv("GRAPHOPPER_KEY")
		if key == "" {
			select {
			case out <- Result[domain.GeoResponse]{Err: fmt.Errorf("env var GRAPHOPPER_KEY not set")}:
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
			case out <- Result[domain.GeoResponse]{Err: fmt.Errorf("error creating request: %w", err)}:
			case <-ctx.Done():
			}
			return
		}

		var geoResponse domain.GeoResponse
		if err := doJSON(client, req, &geoResponse); err != nil {
			out <- Result[domain.GeoResponse]{Err: fmt.Errorf("error doing request: %w", err)}
			return
		}

		select {
		case out <- Result[domain.GeoResponse]{Value: geoResponse}:
		case <-ctx.Done():
			return
		}
	}()
	return out
}
