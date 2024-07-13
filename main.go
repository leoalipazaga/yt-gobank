package main

import (
	"log"

	"github.com/lpernett/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	store, err := NewPostgresStore()
	if err != nil {
		log.Fatalf("there was an error: %v", err)
	}

	if err := store.Init(); err != nil {
		log.Fatal(err)
	}
	server := NewApiServer(":3000", store)
	server.Run()
}
