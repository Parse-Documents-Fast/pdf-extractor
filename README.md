# pdf-extractor

Microservicio en Go que convierte PDFs a Markdown preservando la estructura del documento (fuente, posición, tablas).

Forma parte de una arquitectura de 6 repositorios. Funciona como **consumer de Redis Streams**: recibe PDFs validados de `pdf-main` por `queue:extraction`, los procesa y publica el resultado en `queue:extraction-results`. **No recibe PDFs por HTTP**; el único HTTP es operativo (`/health`, `/metrics`).

> Estado: en desarrollo. Ver el plan y el listado de tasks del repositorio.

## Flujo

```
pdf-main ──► queue:extraction ──► pdf-extractor ──► queue:extraction-results
                                        │
                                        └──► MongoDB (PdfDocument)
```

1. Lee el mensaje JSON (`pdf_id`, `filename`, `content_base64`) desde el consumer group.
2. Decodifica el base64 y extrae bloques estructurados (texto, tamaño de fuente, posición).
3. Mapea los bloques a Markdown.
4. Persiste en MongoDB (`PdfDocument`) y publica el resultado JSON en `queue:extraction-results`.
5. `XACK` al terminar.

## Requisitos

- Go 1.25+
- Redis (con soporte de Streams)
- MongoDB

## Configuración

```bash
cp .env.example .env
```

Editar `.env` con los valores del entorno. Las variables están documentadas en `.env.example`. Al arrancar, la configuración se valida y el proceso falla si falta algo obligatorio (`REDIS_URI`, `MONGO_URI`, `MONGO_DB`, `MONGO_COLLECTION`).

## Construir y ejecutar

```bash
go build -o bin/pdf-extractor ./cmd/pdf-extractor
./bin/pdf-extractor
```

o, sin compilar un binario:

```bash
go run ./cmd/pdf-extractor
```

El proceso se apaga limpio con `SIGINT`/`SIGTERM` (drena los mensajes en vuelo y cierra Redis/MongoDB).

## Tests

```bash
go test ./...
```

Los tests de integración (`repository`, `consumer`) usan [testcontainers-go](https://golang.testcontainers.org/) y requieren Docker. Si el reaper de testcontainers (`ryuk`) no arranca en tu entorno, se pueden correr con:

```bash
TESTCONTAINERS_RYUK_DISABLED=true go test ./...
```

## Estructura

```
cmd/pdf-extractor/   # punto de entrada (wiring + graceful shutdown)
internal/
  config/            # carga y validación de variables de entorno
  consumer/          # consumer de Redis Streams (loop + setup del group)
  dto/               # DTOs JSON del contrato (entrada/salida)
  extractor/         # extracción estructurada del PDF
  markdown/          # mapeo de bloques a Markdown
  models/            # modelo de persistencia (PdfDocument)
  mongodb/           # cliente MongoDB (conexión)
  redis/             # cliente Redis (conexión)
  repository/        # persistencia (MongoDB)
  httpserver/        # /health y /metrics (pendiente)
testdata/            # PDFs de prueba
research/            # spike de elección de librería PDF
docs/                # documentación y ADRs
```

## Documentación

- `docs/`: arquitectura, ADRs y contrato de Redis Streams (pendiente).
- `docs/PDF_LIBRARY_CHOICE.md`: decisión de la librería de extracción (`ledongthuc/pdf`).
