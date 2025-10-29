package main

import (
	"api-parser/internal/application"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load("api-parser.env"); err != nil {
		log.Print("No .env file found")
	}
}

func main() {
	client := &http.Client{Timeout: 10 * time.Second}
	app := application.NewApplication(os.Stdin, os.Stdout, client)
	err := app.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
