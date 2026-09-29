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

1. Lee `pdf_id`, `filename`, `content_base64` desde el consumer group.
2. Decodifica y extrae bloques estructurados (texto, tamaño de fuente, posición, celdas de tabla).
3. Mapea los bloques a Markdown.
4. Persiste en MongoDB y publica el resultado.
5. `XACK` al terminar.

## Requisitos

- Go 1.22+
- Redis (con soporte de Streams)
- MongoDB

## Configuración

```bash
cp .env.example .env
```

Editar `.env` con los valores del entorno. Las variables están documentadas en `.env.example`.

## Estructura

```
cmd/pdf-extractor/   # punto de entrada
internal/
  config/            # carga y validación de variables de entorno
  extractor/         # extracción estructurada del PDF
  markdown/          # mapeo de bloques a Markdown
  repository/        # persistencia (MongoDB)
  consumer/          # consumer de Redis Streams
  httpserver/        # /health y /metrics
testdata/            # PDFs de prueba
docs/                # documentación y ADRs
```

## Desarrollo

Los comandos de build, test y ejecución se documentarán a medida que se agreguen (Makefile en tasks siguientes).

## Documentación

- `docs/`: arquitectura, ADRs y contrato de Redis Streams (pendiente).

