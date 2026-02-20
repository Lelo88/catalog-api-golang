package config

import (
	"fmt"
	"os"
	"strings"
)

// Config agrupa la configuración necesaria para correr la aplicación.
type Config struct {
	Port        string
	DatabaseURL string
}

// Load lee variables de entorno y valida lo mínimo indispensable.
func Load() (Config, error) {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}
	// Normalizamos por si alguien manda ":8080"
	port = strings.TrimPrefix(port, ":")

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return Config{}, fmt.Errorf("missing required env var: DATABASE_URL")
	}

	return Config{
		Port:        port,
		DatabaseURL: databaseURL,
	}, nil
}
