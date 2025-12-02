package main

import (
	"log/slog"
	"os"
	"strconv"
)

func main() {
	args := os.Args[1:]
	port, err := strconv.Atoi(args[0])
	if err != nil {
		slog.Error("invalid port argument: ", err.Error())
		os.Exit(1)
	}
	
}
