# catalog-api-golang

API backend de catálogo en **Go (Golang)** con **PostgreSQL**, migraciones y una suite amplia de tests con cobertura.
Incluye CRUD completo para `items` y utilidades de infraestructura (response helpers, request id, config/db bootstrap).

---

## ¿Qué hace esta API?
Expone endpoints HTTP para administrar un catálogo de items (crear, listar, obtener, actualizar parcialmente y eliminar), con:
- Validaciones de negocio (por ejemplo formato/valor de `price`)
- Respuestas JSON estandarizadas
- Request ID para trazabilidad
- Persistencia en PostgreSQL
- Migraciones versionadas
- Tests por capa con Testify + cobertura

---

## Características
- CRUD de Items:
  - `POST /items`
  - `GET /items`
  - `GET /items/{id}`
  - `PATCH /items/{id}`
  - `DELETE /items/{id}`
- PostgreSQL vía Docker Compose
- Migraciones con `golang-migrate/migrate`
- Respuestas JSON estandarizadas (`data`, `error`, `meta`)
- Request ID para trazabilidad
- Tests con **Testify**:
  - service
  - repository
  - handlers
  - routes
  - middleware/utilities (request_id, response, etc.)
  - config, db, main (bootstrap)
- Comandos `make` para tests, cobertura y entorno local

---

## Requisitos
- **Go** (recomendado: versión reciente)
- **Docker Desktop** (para DB local)
- **make** (opcional pero recomendado)

> Si no querés instalar `migrate` localmente, podés usarlo con Docker (recomendado y multiplataforma).

---

## Configuración

### Variables de entorno
Se utiliza `DATABASE_URL` para conectarse a PostgreSQL.

Ejemplo:
```bash
export DATABASE_URL="postgres://catalog:catalog@localhost:5432/catalog_db?sslmode=disable"

# Si usás .env, recordá exportarlo antes de correr migraciones/tests:
set -a; source .env; set +a

# Levantar la DB:
make db-up

# Ver contenedores:
make db-ps

# Ver logs:
make db-logs

# Bajar db: 
make db-down

# Aplicar migraciones:
docker run --rm -v "$(pwd)/migrations:/migrations" \
  migrate/migrate \
  -path=/migrations \
  -database "$DATABASE_URL" up

# Ver versión de migraciones:
docker run --rm -v "$(pwd)/migrations:/migrations" \
  migrate/migrate \
  -path=/migrations \
  -database "$DATABASE_URL" version

# Crear migración:
docker run --rm -v "$(pwd)/migrations:/migrations" \
  migrate/migrate create -ext sql -dir /migrations -seq nombre_de_migracion


# Instalar migrate localmente en mac:
brew install golang-migrate

# Instalar migrate localmente en linux:
tar -xvf migrate.linux-amd64.tar.gz
sudo mv migrate /usr/local/bin/migrate
migrate -version


# Instalar migrate localmente en windows:
choco install migrate

# Aplicar migraciones con make 
make migrate-up

# Ver versión de migraciones con make
make migrate-version

# Correr la API con make
make run

# Correr tests con make
make test

# Cobertura con make
make cover

# HTML con make
make cover-html

# Funciones con make
make cover-func

Ejemplos (curl); 

# Crear item

curl -X POST http://localhost:8080/items \
  -H 'Content-Type: application/json' \
  -d '{"name":"Product","price":"1000.00","stock":2}'

# Listar items

curl "http://localhost:8080/items?page=1&limit=10&query=prod"

# Obtener item por ID
curl http://localhost:8080/items/{id}

# Actualizar item parcialmente
curl -X PATCH http://localhost:8080/items/{id} \
 -H 'Content-Type: application/json' \
 -d '{"stock": 10}'

# Actualizar a null
curl -X PATCH http://localhost:8080/items/{id} \
 -H 'Content-Type: application/json' \
 -d '{"description": null}'

# Eliminar item
curl -X DELETE http://localhost:8080/items/{id}

## Documentación (OpenAPI / Swagger)

Este proyecto expone documentación interactiva usando **Swagger UI** y el contrato **OpenAPI**.

### Abrir Swagger UI
1. Levantar la API:
```bash
make run
```
2. Abrir en el navegador:
```
http://localhost:8080/docs/
```

### Ver el contrato OpenAPI
```
http://localhost:8080/openapi.yaml
```

