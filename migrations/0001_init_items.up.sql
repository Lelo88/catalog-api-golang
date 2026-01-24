-- Migración inicial: tabla items.
-- Usamos UUID como PK para evitar ids secuenciales y simplificar futuros escenarios distribuidos.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS items (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  description text,
  price numeric(10,2) NOT NULL,
  stock integer NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),

  -- Reglas mínimas a nivel DB (no reemplazan validación de service, la refuerzan).
  CONSTRAINT ck_items_price_positive CHECK (price > 0),
  CONSTRAINT ck_items_stock_non_negative CHECK (stock >= 0)
);

-- Para evitar duplicados obvios en catálogo (ajustable según tu criterio).
CREATE UNIQUE INDEX IF NOT EXISTS ux_items_name ON items (name);

-- Índice para búsquedas por nombre. En el futuro, si querés búsqueda parcial, se puede mejorar con trigram.
CREATE INDEX IF NOT EXISTS ix_items_name ON items (name);
