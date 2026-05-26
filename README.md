# Microservicio de Consumos de Energía

Microservicio en Go que expone un endpoint para consultar consumos de energía de medidores, agregados por día, semana o mes, a partir de datos almacenados en PostgreSQL.

## Tecnologías

- Go (Golang)
- Gin (framework HTTP)
- PostgreSQL
- Driver Postgres: `github.com/lib/pq`
- Tests con `testing` (Go estándar)

## Requisitos

- Go instalado (>= 1.20)
- PostgreSQL instalado y ejecutándose
- Git (para clonar el repositorio)

## Configuración de base de datos

1. Crear base de datos:

```sql
CREATE DATABASE energy_db;
Conectarse a energy_db y crear la tabla:

CREATE TABLE consumptions (
    id                  UUID PRIMARY KEY,
    meter_id            INT NOT NULL,
    active              NUMERIC(18,6) NOT NULL DEFAULT 0,
    reactive_inductive  NUMERIC(18,6) NOT NULL DEFAULT 0,
    reactive_capacitive NUMERIC(18,6) NOT NULL DEFAULT 0,
    exported            NUMERIC(18,6) NOT NULL DEFAULT 0,
    measured_at         TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_consumptions_meter_date
  ON consumptions(meter_id, measured_at);

Importar el CSV de consumos en la tabla consumptions con pgAdmin o COPY, en el orden:

id, meter_id, active, measured_at

##**Configuración del servicio**##

En main.go se configura el DSN de Postgres:

const dsn = "postgres://postgres:<TU_PASSWORD>@localhost:5432/energy_db?sslmode=disable"
Cambiar <TU_PASSWORD> por la contraseña real del usuario de PostgreSQL (y el usuario/host/puerto si aplica).

##**Ejecutar el servicio**##
go run .

Por defecto el servicio escucha en el puerto 8080.

##**Endpoint de prueba**##

GET /ping
Respuesta:

{"message": "pong"}

##**Endpoint principal**##

##**GET /consumption**##

Obtiene consumos agregados por periodo (daily, weekly, monthly).

Query params:

meters_ids (string, requerido): lista de IDs de medidor, separados por coma.
Ej: 1 o 1,2,3
start_date (string, requerido): fecha de inicio YYYY-MM-DD.
Ej: 2023-06-01
end_date (string, requerido): fecha de fin YYYY-MM-DD.
Ej: 2023-06-10
kind_period (string, requerido): daily, weekly o monthly.
Ejemplos:

##**Diario:**##
curl "http://localhost:8080/consumption?meters_ids=1&start_date=2023-06-01&end_date=2023-06-10&kind_period=daily"

##**Semanal:**##
curl "http://localhost:8080/consumption?meters_ids=1&start_date=2023-06-01&end_date=2023-06-26&kind_period=weekly"

##**Mensual:**##
curl "http://localhost:8080/consumption?meters_ids=1&start_date=2023-06-01&end_date=2023-07-10&kind_period=monthly"

##**Respuesta (estructura):**##
{
  "period": ["JUN 1", "JUN 2", "..."],
  "data_graph": [
    {
      "meter_id": 1,
      "address": "Dirección mock",
      "active": [0, 0, ...],
      "reactive_inductive": [0, 0, ...],
      "reactive_capacitive": [0, 0, ...],
      "exported": [0, 0, ...]
    }
  ]
}

period: etiquetas de los periodos (día, semana o mes).
data_graph: un elemento por medidor solicitado.
Los arrays active, reactive_inductive, reactive_capacitive, exported tienen la misma longitud que period.
Nota: En esta versión, address se retorna como "Dirección mock" para simular el microservicio de direcciones.

##**Tests**##
Se incluyen tests unitarios para la función buildPeriods (generación de periodos diarios, semanales y mensuales) en el archivo:

  period_test.go

Para ejecutar los tests:
go test ./...

Para más detalles, ver el documento de documentación técnica incluido en el repositorio (BIA_documentación.pdf / .docx)
