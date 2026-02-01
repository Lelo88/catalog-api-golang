package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoad_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("PORT", "")

	cfg, err := Load()

	require.Error(t, err)
	require.Equal(t, Config{}, cfg)
}

func TestLoad_DefaultPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("PORT", "")

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, "8080", cfg.Port)
	require.Equal(t, "postgres://example", cfg.DatabaseURL)
}

func TestLoad_CustomPort(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("PORT", "9090")

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, "9090", cfg.Port)
	require.Equal(t, "postgres://example", cfg.DatabaseURL)
}
