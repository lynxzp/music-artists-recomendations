package main

import (
	"log"
	"music-recomendations/internal/service"
)

func main() {
	err := service.Run()
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("service gracefully stopped")
	}

}
