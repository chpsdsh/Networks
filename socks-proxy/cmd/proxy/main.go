package main

import (
	"log/slog"
	"os"
	"socks-proxy/internal/application"
	"strconv"
)

func main() {
	args := os.Args[1:]
	port, err := strconv.Atoi(args[0])
	if err != nil {
		slog.Error("invalid port argument: ", err.Error())
		os.Exit(1)
	}
	server, err := application.NewServer(port)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	if err := server.Start(); err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
