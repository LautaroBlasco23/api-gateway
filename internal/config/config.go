package config

import "os"

type Config struct {
	Port         string
	RegistryFile string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	registryFile := os.Getenv("REGISTRY_FILE")
	if registryFile == "" {
		registryFile = "registry.json"
	}
	return &Config{Port: port, RegistryFile: registryFile}
}
