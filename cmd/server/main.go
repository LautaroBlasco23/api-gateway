package main

import (
	"log"
	"net/http"

	"api-gateway/internal/config"
	"api-gateway/internal/gateway"
	"api-gateway/internal/registry"
)

func main() {
	cfg := config.Load()
	reg := registry.New()
	router := gateway.NewRouter(reg)

	log.Printf("API Gateway listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, router); err != nil {
		log.Fatal(err)
	}
}
