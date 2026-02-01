# Makefile - tareas frecuentes (tests, cobertura, db, migraciones)
# Requiere:
# - Go instalado
# - Docker Desktop
# - migrate (golang-migrate CLI) para migraciones (opcional si usás migrate con Docker)
#
# Tips:
# - Cobertura por paquete en terminal: make cover
# - Perfil y detalle por funciones: make cover-func

APP_NAME ?= catalog-api
PKG ?= ./...
COVER_FILE ?= coverage.out

DB_URL ?= $(DATABASE_URL)

.PHONY: help docker-check db-up db-down db-logs db-ps \
        migrate-up migrate-down migrate-version migrate-create \
        test cover cover-func cover-html it run tidy fmt

help:
	@echo ""
	@echo "Targets:"
	@echo "  make test         - corre tests de todos los paquetes"
	@echo "  make cover        - cobertura por paquete (terminal)"
	@echo "  make cover-func   - coverprofile + cobertura por función"
	@echo "  make cover-html   - reporte HTML de cobertura"
	@echo "  make db-up        - levanta postgres con docker compose"
	@echo "  make db-down      - baja postgres"
	@echo "  make migrate-up   - aplica migraciones (requiere DATABASE_URL)"
	@echo "  make it           - integración (db-up + migrate-up + tags=integration)"
	@echo "  make run          - corre la API"
	@echo ""

docker-check:
	@docker info >/dev/null 2>&1 || (echo "Docker no responde. Abrí Docker Desktop y reintentá."; exit 1)

db-up: docker-check
	docker compose up -d

db-down: docker-check
	docker compose down

db-logs: docker-check
	docker compose logs -f

db-ps: docker-check
	docker compose ps

migrate-up:
	@if [ -z "$(DB_URL)" ]; then echo "DATABASE_URL no está seteada"; exit 1; fi
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	@if [ -z "$(DB_URL)" ]; then echo "DATABASE_URL no está seteada"; exit 1; fi
	migrate -path migrations -database "$(DB_URL)" down 1

migrate-version:
	@if [ -z "$(DB_URL)" ]; then echo "DATABASE_URL no está seteada"; exit 1; fi
	migrate -path migrations -database "$(DB_URL)" version

# Uso: make migrate-create name=agregar_tabla_x
migrate-create:
	@if [ -z "$(name)" ]; then echo "Falta name. Ej: make migrate-create name=agregar_tabla_x"; exit 1; fi
	migrate create -ext sql -dir migrations -seq "$(name)"

test:
	go test $(PKG) -count=1

# Cobertura por paquete (lo que pediste: muestra % por paquete en terminal)
cover:
	go test $(PKG) -cover -count=1

# Perfil de cobertura + detalle por función/archivo.
cover-func:
	go test $(PKG) -coverprofile=$(COVER_FILE) -covermode=atomic -count=1
	go tool cover -func=$(COVER_FILE)

# Reporte HTML (abre en macOS)
cover-html:
	go test $(PKG) -coverprofile=$(COVER_FILE) -covermode=atomic -count=1
	go tool cover -html=$(COVER_FILE) -o coverage.html
	@command -v open >/dev/null 2>&1 && open coverage.html || echo "Generado: coverage.html"

# Tests de integración (si existen tests con build tag integration)
it: db-up migrate-up
	go test -tags=integration $(PKG) -count=1

run:
	go run ./cmd/api

tidy:
	go mod tidy

fmt:
	go fmt ./...

openapi-sync:
	cp docs/openapi.yaml internal/docs/openapi.yaml

OPENAPI_FILE ?= docs/openapi.yaml

openapi-validate:
	npx -y @redocly/cli lint docs/openapi.yaml
