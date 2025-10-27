package application

import (
	"api-parser/internal/domain"
	"api-parser/internal/infrastructure/console"
	"api-parser/internal/infrastructure/network"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

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
	err := app.printer.Print("Введите название места")
	if err != nil {
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
	for _, l := range locOut.Value {
		err := app.printer.Print(l)
		if err != nil {
			return err
		}
	}
	targetLocation, err := app.chooseLocation(locOut.Value)
	if err != nil {
		return err
	}
	ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel1()
	weatherOut := network.ReadWeatherDataAsync(ctx1, app.client, targetLocation.Lat, targetLocation.Lng)
	data := <-weatherOut
	fmt.Println(data)
	return nil
}

func (app Application) chooseLocation(locations []domain.Location) (domain.Location, error) {
	for {
		locId, err := app.input.InputData()
		if err != nil {
			return domain.Location{}, err
		}
		for _, l := range locations {
			id, err := strconv.ParseInt(locId, 10, 64)
			if err != nil {
				return domain.Location{}, err
			}
			if int(id) == l.Id {
				return l, nil
			}
		}
	}
}
