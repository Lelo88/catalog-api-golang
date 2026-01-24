# Makefile para tareas frecuentes en desarrollo.
# Requiere:
# - Docker (para levantar postgres local)
# - migrate (CLI de golang-migrate)

DB_URL ?= $(DATABASE_URL)

.PHONY: migrate-up migrate-down migrate-version migrate-create

# Aplica todas las migraciones pendientes.
migrate-up:
	@if [ -z "$(DB_URL)" ]; then echo "DATABASE_URL no está seteada"; exit 1; fi
	migrate -path migrations -database "$(DB_URL)" up

# Revierte la última migración aplicada.
migrate-down:
	@if [ -z "$(DB_URL)" ]; then echo "DATABASE_URL no está seteada"; exit 1; fi
	migrate -path migrations -database "$(DB_URL)" down 1

# Muestra la versión actual de la DB.
migrate-version:
	@if [ -z "$(DB_URL)" ]; then echo "DATABASE_URL no está seteada"; exit 1; fi
	migrate -path migrations -database "$(DB_URL)" version

# Crea un nuevo par de migraciones (up/down).
# Uso: make migrate-create name=agregar_tabla_x
migrate-create:
	@if [ -z "$(name)" ]; then echo "Falta name. Ej: make migrate-create name=agregar_tabla_x"; exit 1; fi
	migrate create -ext sql -dir migrations -seq "$(name)"
