package config

import (
	"fmt"
	"os"
)

// Config agrupa la configuración necesaria para correr la aplicación.
// Se carga desde variables de entorno para facilitar despliegues (Render, Railway, etc.).
type Config struct {
	Port        string
	DatabaseURL string
}

// Load lee variables de entorno y valida lo mínimo indispensable.
// Nota: preferimos fallar rápido al iniciar si falta DATABASE_URL, para evitar errores silenciosos.
func Load() (Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return Config{}, fmt.Errorf("missing required env var: DATABASE_URL")
	}

	return Config{
		Port:        port,
		DatabaseURL: dbURL,
	}, nil
}

