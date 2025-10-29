package network

import (
	"api-parser/internal/domain"
	"api-parser/internal/infrastructure/network/utils"
	"context"
	"fmt"
	"net/http"
	"os"
)

func ReadWeatherDataAsync(ctx context.Context, client *http.Client, Lat, Lng float64) chan utils.Result[domain.WeatherResponse] {
	out := make(chan utils.Result[domain.WeatherResponse], 1)
	go func() {
		defer close(out)

		key := os.Getenv("OPEN_WEATHER_KEY")
		if key == "" {
			select {
			case out <- utils.Result[domain.WeatherResponse]{Err: fmt.Errorf("OPEN_WEATHER_KEY not found in env")}:
			case <-ctx.Done():
			}
			return
		}

		q := fmt.Sprintf(
			"https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&appid=%s",
			Lat, Lng, key)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, q, nil)
		if err != nil {
			select {
			case out <- utils.Result[domain.WeatherResponse]{Err: err}:
			case <-ctx.Done():
			}
			return
		}
		var weatherResp domain.WeatherResponse
		if err := utils.DoJSON(client, req, &weatherResp); err != nil {
			select {
			case out <- utils.Result[domain.WeatherResponse]{Err: err}:
			case <-ctx.Done():
			}
			return
		}

		select {
		case out <- utils.Result[domain.WeatherResponse]{Value: weatherResp}:
		case <-ctx.Done():
		}
	}()
	return out
}
