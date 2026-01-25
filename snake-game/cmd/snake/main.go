package main

import (
	"log"
	"log/slog"
	"os"
	"snake-game/internal/application/client"
	"snake-game/internal/infrastructure/gui"
	"snake-game/internal/infrastructure/network/transport"

	fyneapp "fyne.io/fyne/v2/app"
)

func main() {
	app := fyneapp.New()
	window := app.NewWindow("Snake game")

	// клиент отвечающий за логику игры и сеть
	gameClient := client.NewGameClient()

	// Создание двух транспортных сокетов
	tr, err := transport.NewTransport(gameClient, "239.192.0.4", 9192)
	if err != nil {
		log.Fatal(err)
	}

	// Связываем клиент с сетью
	gameClient.SetTransport(tr)

	// Создаём и подключаем GUI
	view := gui.NewFyneView(app, window, gameClient)
	gameClient.SetView(view)

	logger := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level:     slog.LevelInfo,
			AddSource: true,
		}),
	)

	slog.SetDefault(logger)

	// Запуск клиента
	gameClient.Run()

	// Запуск GUI-цикла
	window.ShowAndRun()
}
