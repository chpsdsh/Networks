package application

import (
	"api-parser/internal/domain"
	"api-parser/internal/infrastructure/console"
	"api-parser/internal/infrastructure/network"
	"context"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

const NumChannels = 2

type Input interface {
	InputData() (string, error)
}
type Printer interface {
	Print(s any) error
}
type Application struct {
	input   Input
	printer Printer
	client  *http.Client
}

func NewApplication(r io.Reader, w io.Writer, client *http.Client) *Application {
	return &Application{input: console.NewConsoleInput(r), printer: console.NewConsoleOutput(w), client: client}
}

func (app Application) Run() error {
	if err := app.printer.Print("Введите название места"); err != nil {
		return err
	}
	place, err := app.input.InputData()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out := network.ReadGeoDataAsync(ctx, app.client, place)
	locOut := <-out
	if locOut.Err != nil {
		return locOut.Err
	}
	for _, l := range locOut.Value.Hits {
		if err := app.printer.Print(l); err != nil {
			return err
		}
	}
	point, err := app.chooseLocation(locOut.Value)
	if err != nil {
		return err
	}
	if err := app.getWeatherAndPlacesInfo(point); err != nil {
		return err
	}
	return nil
}

func (app Application) getWeatherAndPlacesInfo(point domain.Point) error {
	weatherCtx, weatherCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer weatherCancel()
	weatherOut := network.ReadWeatherDataAsync(weatherCtx, app.client, point.Lat, point.Lng)

	placesInfoCtx, placesInfoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer placesInfoCancel()
	wikiOut := network.ReadWikiDataAsync(placesInfoCtx, app.client, point.Lat, point.Lng)
	interPlaces := network.ReadPlaceDescriptionAsync(placesInfoCtx, app.client, wikiOut)

	if err := app.getResults(weatherOut, interPlaces, weatherCtx, placesInfoCtx); err != nil {
		return err
	}
	return nil
}

func (app Application) getResults(weatherOut <-chan network.Result[domain.WeatherResponse], interPlaces <-chan network.Result[[]domain.PlaceInfo], weatherCtx context.Context, placesInfoCtx context.Context) error {
	wch := weatherOut
	pch := interPlaces
	for wch != nil || pch != nil {
		select {
		case res, ok := <-wch:
			if !ok {
				wch = nil
				log.Println("wch is closed")
				continue
			}
			if res.Err != nil {
				return res.Err
			}
			if err := app.printer.Print(res.Value); err != nil {
				return err
			}
		case res, ok := <-pch:
			if !ok {
				pch = nil
				log.Println("pch is closed")

				continue
			}
			if res.Err != nil {
				return res.Err
			}
			if err := app.printer.Print(res.Value); err != nil {
				return err
			}
		case <-weatherCtx.Done():
			return weatherCtx.Err()
		case <-placesInfoCtx.Done():
			return placesInfoCtx.Err()
		}
	}
	return nil
}

func (app Application) chooseLocation(locations domain.GeoResponse) (domain.Point, error) {
	for {
		locId, err := app.input.InputData()
		if err != nil {
			return domain.Point{}, err
		}
		for _, l := range locations.Hits {
			id, err := strconv.ParseInt(locId, 10, 64)
			if err != nil {
				return domain.Point{}, err
			}
			if int(id) == l.OSMId {
				return domain.Point{Lat: l.Point.Lat, Lng: l.Point.Lng}, nil
			}
		}
	}
}
