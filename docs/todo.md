# TODO — PDF Extractor (Go)

## Setup Inicial
- [ ] `go mod init` con el path del módulo de la organización
- [ ] Estructura base: `cmd/pdf-extractor/`, `internal/{config,extractor,markdown,repository,consumer,httpserver}`, `testdata/`, `docs/`
- [ ] Dependencias: `go-redis/v9`, `mongo-driver`, `prometheus/client_golang`, `testify`, `testcontainers-go`
- [ ] `Makefile` (`build`, `test`, `lint`, `run`)
- [ ] `.golangci.yml`
- [ ] `.env.example` (REDIS_URI, MONGO_URI, MONGO_DB, CONSUMER_NAME, HTTP_ADDR, umbrales, timeouts)
- [ ] `internal/config` con validación al arrancar
- [ ] Logging con `log/slog` (JSON)
- [ ] `main.go` con graceful shutdown (`signal.NotifyContext`)
- [ ] `Dockerfile` multi-stage
- [ ] CI: `go vet`, lint, tests

## Consumer Base (con procesador stub)
- [ ] Cliente Redis + creación del consumer group `pdf-extractor-group` (`XGROUP CREATE ... MKSTREAM`, tolerar `BUSYGROUP`)
- [ ] Loop `XREADGROUP` sobre `queue:extraction` que solo loguea el mensaje
- [ ] Apagado limpio verificado

## Librería PDF (spike)
- [ ] Preparar PDFs de prueba en `testdata/` (simple, con tabla, corrupto, sin texto extraíble)
- [ ] Revisar licencias: `unipdf` (comercial/AGPL), `pdfcpu` (Apache), fork de `rsc.io/pdf`, `go-pdfium`/`go-fitz`
- [ ] Prototipar candidatos y comparar (fuente, posición, robustez, cgo, mantenimiento)
- [ ] Elegir y documentar la decisión en un ADR (`docs/`)

## Extracción de Estructura
- [ ] Tipo `Block` (texto, tamaño de fuente, posición X/Y, ancho, página, `IsTableCell`)
- [ ] `ExtractStructure(ctx, pdf []byte) ([]Block, error)`
- [ ] Detección de tablas por alineación de coordenadas
- [ ] `PdfExtractionError` + sentinelas (`ErrCorruptPDF`, `ErrNoExtractableText`) compatibles con `errors.Is/As`
- [ ] `recover()` para panics de la librería con PDFs malformados
- [ ] Respetar `ctx` (timeout por documento)
- [ ] Casos borde: corrupto, sin texto, con contraseña, muy grande
- [ ] Tests con PDF simple + tabla

## Mapeo a Markdown
- [ ] `MapStructureToMarkdown(blocks []Block) string`
- [ ] Reglas: tamaño de fuente > umbral → headers, tablas → sintaxis MD, resto → párrafo
- [ ] Umbrales relativos al tamaño de fuente predominante y configurables
- [ ] Escapar caracteres especiales de Markdown y `|` en celdas
- [ ] Ajustar umbrales iterativamente con PDFs reales
- [ ] Tests table-driven + golden files

## Persistencia MongoDB
- [ ] Struct `PdfDocument` con tags `bson`: `pdf_id`, `filename`, `markdown_content`, `extracted_at`, `status`, `error`
- [ ] Interfaz `PdfRepository`
- [ ] `MongoPdfRepository` con `upsert` por `pdf_id` (idempotente)
- [ ] Índice único en `pdf_id`
- [ ] Tests de integración con `testcontainers-go`

## Redis Streams Consumer (pipeline real)
- [ ] Parser del mensaje: `pdf_id`, `filename`, `content_base64` (validar campos y tamaño máximo)
- [ ] Decodificar base64 → bytes
- [ ] Procesar (Extract + Map) y persistir en Mongo
- [ ] Publicar resultado en `queue:extraction-results` (éxito o `failed`)
- [ ] `XACK` solo después de persistir y publicar
- [ ] Reintentos: `XAUTOCLAIM` para pendientes + conteo de entregas con `XPENDING`
- [ ] Clasificar errores: permanentes (sin reintento) vs transitorios (backoff)
- [ ] Dead-letter queue `queue:extraction-dlq` tras N intentos
- [ ] Worker pool con concurrencia configurable
- [ ] Drenado de mensajes en vuelo durante shutdown

## Health Check + Metrics
- [ ] `GET /health`
- [ ] `GET /ready` (verifica Redis y Mongo)
- [ ] `GET /metrics` con `promhttp`
- [ ] Métricas: procesados por `status`, errores por tipo, latencia (histograma), DLQ, pendientes
- [ ] Servidor HTTP con timeouts y shutdown ordenado

## Tests
- [ ] Unitarios: `ExtractStructure`, `MapStructureToMarkdown`, parser de mensajes
- [ ] Integración: consumer + Redis + Mongo (reintento y DLQ incluidos)
- [ ] Fixtures en `testdata/`: PDFs simple + tabla + corrupto + sin texto
- [ ] Benchmarks del pipeline
- [ ] Umbral de cobertura en CI

## Documentación
- [ ] `README.md` con setup, envs, cómo correr y testear
- [ ] `.env.example` con todas las variables
- [ ] Architecture docs en `/docs`
- [ ] Contrato Redis Streams documentado (mensaje de entrada, resultado, DLQ)
- [ ] ADR de elección de librería PDF

## Reconciliación entre servicios
- [ ] Confirmar host/puerto de Redis en la red interna
- [ ] Confirmar host/puerto/credenciales de MongoDB y base/colección de `PdfDocument`
- [ ] Confirmar nombres de streams (`queue:extraction`, `queue:extraction-results`) y proponer `queue:extraction-dlq`
- [ ] Confirmar nombre del consumer group
- [ ] Confirmar formato exacto del mensaje de entrada en `pdf-main`
- [ ] Definir formato del mensaje de resultado con el consumidor
- [ ] Confirmar tamaño máximo de PDF
- [ ] Confirmar puerto de `/health` y `/metrics` y el scrape de Prometheus
- [ ] Actualizar `.env.example` y el contrato en `docs/` con los valores confirmados