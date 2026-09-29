# Plan de Desarrollo — PDF Extractor Microservice (Go)

## Contexto
Microservicio en **Go** que absorbe responsabilidades de `pdf-transformator` (ADR-0005). Funciona como **consumer de Redis Streams**. Solo expone HTTP para operación (`/health`, `/metrics`), nunca para recibir PDFs. Recibe PDFs validados de `pdf-main` vía `queue:extraction` y publica resultados en `queue:extraction-results`.

## Decisión Clave
Nunca aplanar el PDF a texto plano antes de tiempo. Mantener estructura (fuente, posición, tabla) para mapear confiablemente a Markdown en vez de heurísticas sobre texto ya aplanado.

## Stack

| Área | Elección |
|---|---|
| Lenguaje | Go (última versión estable, fijada en `go.mod`) |
| HTTP (health/metrics) | `net/http` estándar (Go 1.22+ soporta métodos en el mux) |
| Redis Streams | `github.com/redis/go-redis/v9` |
| MongoDB | `go.mongodb.org/mongo-driver` (driver oficial; no hay ODM equivalente a Beanie, se usa el driver directo detrás de `PdfRepository`) |
| Métricas | `github.com/prometheus/client_golang` |
| Config | Variables de entorno (`caarlos0/env` o `os.Getenv` + validación propia) |
| Logging | `log/slog` (estándar), en JSON |
| Tests | `testing` + `testify`; `testcontainers-go` para Redis y MongoDB |
| Lint | `golangci-lint` |
| Librería PDF | **A definir en el spike de Fase 2** (ver abajo) |

## Estructura del repo

```
pdf-extractor/
├── cmd/pdf-extractor/main.go     # wiring, graceful shutdown
├── internal/
│   ├── config/                   # carga y validación de env vars
│   ├── extractor/                # Block, ExtractStructure, PdfExtractionError
│   ├── markdown/                 # MapStructureToMarkdown
│   ├── repository/               # PdfDocument, PdfRepository, MongoPdfRepository
│   ├── consumer/                 # consumer group, reintentos, DLQ, procesamiento
│   └── httpserver/               # /health, /metrics
├── testdata/                     # PDFs de prueba (simple, con tabla, corrupto, sin texto)
├── docs/
├── Dockerfile
├── Makefile
├── go.mod
├── .env.example
├── .golangci.yml
└── README.md
```

> Se usa `internal/` para que nada del servicio sea importable desde afuera. Si la organización exige otro layout (p. ej. `dev/`), se adapta antes de la Fase 1.

---

## Fase 1: Estructura Base + Consumer Loop (con procesador stub)

- [ ] `go mod init` con el path del módulo de la organización
- [ ] Crear estructura de carpetas (ver arriba), `Makefile` (`build`, `test`, `lint`, `run`), `.golangci.yml`
- [ ] `.env.example` y `README.md` inicial
- [ ] `internal/config`: struct tipado con env vars (Redis URI, Mongo URI, nombre de DB, nombre del consumer, addr HTTP, umbrales, timeouts); falla al arrancar si falta algo obligatorio
- [ ] `log/slog` en JSON
- [ ] `main.go` con `context` + `signal.NotifyContext` para graceful shutdown (SIGTERM/SIGINT)
- [ ] Cliente `go-redis`, creación del consumer group `pdf-extractor-group` (`XGROUP CREATE ... MKSTREAM`, tolerando `BUSYGROUP`)
- [ ] Consumer loop con `XREADGROUP` que lee `queue:extraction` y procesa con un **stub** (solo loguea)
- [ ] `Dockerfile` multi-stage y pipeline de CI (`go vet`, lint, test)

**Listo cuando:** el servicio arranca, se une al grupo, consume un mensaje de prueba y se apaga limpio.

---

## Fase 2: Extracción Estructurada de PDF

### Spike: elegir librería
Candidatos (verificar licencia y capacidades reales en el spike):

- **`unipdf`**: potente, pero **licencia comercial/AGPL con API key**. Riesgo alto para un repo de organización, validar con legal antes de usarla.
- **`pdfcpu`**: Apache 2.0, muy buena para manipular PDFs (merge, validate, etc.), pero **no está pensada para extracción de texto con posición/fuente**. Probablemente insuficiente.
- **Fork mantenido de `rsc.io/pdf`** (p. ej. `dslipak/pdf`): BSD, expone texto con fuente, tamaño y coordenadas (X, Y). Candidato fuerte para este caso.
- **`go-pdfium` / `go-fitz` (MuPDF)**: más robustos, pero traen dependencia nativa (cgo/wasm) y MuPDF es AGPL. Evaluar impacto en build e imagen Docker.

Criterios: licencia compatible, acceso a tamaño de fuente y posición, robustez con PDFs corruptos, sin cgo si es posible, mantenimiento activo.

- [ ] Preparar PDFs de prueba en `testdata/` (simple, con tabla, corrupto, sin texto extraíble)
- [ ] Prototipo por candidato y tabla comparativa
- [ ] Registrar la decisión en un ADR en `docs/`

### Implementación
- [ ] Tipo `Block`: `Text`, `FontSize`, `X`, `Y`, `Width`, `Page`, `IsTableCell` (más `Row`/`Col` si la detección de tablas lo requiere)
- [ ] `ExtractStructure(ctx context.Context, pdf []byte) ([]Block, error)`
- [ ] Detección de tablas: agrupar por alineación de coordenadas (filas por Y, columnas por X). Ninguna librería lo da hecho, es heurística propia sobre posiciones
- [ ] Errores de dominio: `PdfExtractionError` como tipo de error con `Unwrap()`, y sentinelas (`ErrCorruptPDF`, `ErrNoExtractableText`) usables con `errors.Is/As`
- [ ] **`recover()` alrededor de la librería**: varias librerías Go de PDF hacen `panic` con archivos malformados; un PDF malo no debe tumbar el consumer
- [ ] Respetar `ctx` (timeout por documento)
- [ ] Casos borde: PDF corrupto, sin texto extraíble (escaneado), PDF protegido con contraseña, PDF muy grande

---

## Fase 3: Mapeo a Markdown

- [ ] `MapStructureToMarkdown(blocks []Block) string`:
  - Tamaño de fuente > umbral → `#` / `##` / `###`
  - Bloques de tabla → sintaxis Markdown (`|...|` con fila separadora)
  - Resto → párrafo plano
- [ ] Umbrales configurables (relativos al tamaño de fuente **predominante** del documento, no valores absolutos)
- [ ] Escapar caracteres especiales de Markdown y `|` dentro de celdas
- [ ] Ajustar umbrales iterando con PDFs reales
- [ ] Tests table-driven + golden files (`testdata/*.md` esperado) contra PDFs simple y con tabla

---

## Fase 4: Persistencia en MongoDB

- [ ] Modelo `PdfDocument` (structs con tags `bson`):
  - `PdfID`, `Filename`, `MarkdownContent`, `ExtractedAt`, `Status` (`success`/`failed`), y `Error` opcional
- [ ] Interfaz `PdfRepository` (definida donde se consume) + `MongoPdfRepository`
- [ ] **Escritura idempotente**: `upsert` por `pdf_id` (Redis Streams es at-least-once, un reintento no debe duplicar)
- [ ] Índice único en `pdf_id`
- [ ] Timeouts vía `context`, cliente Mongo con cierre en shutdown
- [ ] Tests de integración con `testcontainers-go`

---

## Fase 5: Integración completa con Redis Streams

Reemplaza el stub de la Fase 1 por el pipeline real.

- [ ] Parser del mensaje: `pdf_id`, `filename`, `content_base64` (validar campos y límite de tamaño)
- [ ] Decodificar base64 → bytes
- [ ] Procesar (`ExtractStructure` + `MapStructureToMarkdown`) → guardar en Mongo
- [ ] Publicar en `queue:extraction-results` con el Markdown resultado (o el error, con `status`)
- [ ] `XACK` **solo después** de persistir y publicar
- [ ] Reintentos: recuperar mensajes pendientes con `XAUTOCLAIM`, contar entregas con `XPENDING`
- [ ] Errores permanentes (PDF corrupto, sin texto, mensaje inválido) → sin reintento, publicar resultado `failed`
- [ ] Errores transitorios (Redis/Mongo caídos) → reintento con backoff hasta N intentos
- [ ] Superado el máximo de intentos → dead-letter queue (`queue:extraction-dlq`) y `XACK` del original
- [ ] Concurrencia configurable (worker pool con N goroutines) y drenado limpio en shutdown

---

## Fase 6: Endpoints Secundarios (Health Check + Metrics)

- [ ] `GET /health` (liveness) y, si aplica, `GET /ready` que verifica Redis y Mongo
- [ ] `GET /metrics` con `promhttp`
- [ ] Métricas: PDFs procesados (por `status`), errores (por tipo), latencia (histograma), mensajes en DLQ, mensajes pendientes
- [ ] Servidor HTTP con timeouts y shutdown ordenado

---

## Fase 7: Tests + Documentación

- [ ] Tests unitarios: `ExtractStructure`, `MapStructureToMarkdown`, parser de mensajes
- [ ] Tests de integración: consumer loop + Redis + Mongo (`testcontainers-go`), incluyendo reintento y DLQ
- [ ] Fixtures en `testdata/`: PDFs simple + tabla + corrupto + sin texto
- [ ] Benchmarks básicos (`go test -bench`) del pipeline
- [ ] Cobertura y CI verdes
- [ ] README con setup, envs, cómo correr y testear
- [ ] Architecture docs en `/docs`, incluyendo el **contrato de Redis Streams** (formato de mensajes de entrada, resultado, DLQ)

---

## Reconciliación entre servicios (pendiente)

Las specs de los repos no se armaron en conjunto. Antes de la Fase 5 hay que confirmar estos valores con los demás repos (sobre todo `pdf-main`) y anotarlos. Todo se configura por variables de entorno, así que un cambio no implica tocar código, salvo el formato de mensajes.

| Ítem | Valor propuesto | Confirmado |
|---|---|---|
| Host/puerto de Redis (red interna) | `redis://localhost:6379/0` | ☐ |
| Host/puerto/credenciales de MongoDB | `mongodb://localhost:27017` | ☐ |
| Base y colección de `PdfDocument` (¿otro servicio escribe ahí?) | `pdf_extractor` / `pdf_documents` | ☐ |
| Stream de entrada | `queue:extraction` | ☐ |
| Stream de resultados | `queue:extraction-results` | ☐ |
| Stream DLQ (propuesto por este servicio) | `queue:extraction-dlq` | ☐ |
| Consumer group | `pdf-extractor-group` | ☐ |
| Formato del mensaje de entrada (campos planos vs JSON en un campo) | campos planos: `pdf_id`, `filename`, `content_base64` | ☐ |
| Formato del mensaje de resultado que espera el consumidor | a definir | ☐ |
| Tamaño máximo de PDF que envía `pdf-main` | 20 MiB (`PDF_MAX_BYTES`) | ☐ |
| Puerto interno de `/health` y `/metrics` / scrape de Prometheus | `:8080` | ☐ |

> El extractor no llama por HTTP a otros servicios; los paths HTTP internos del validador y el convertidor no le aplican.

---

## Ya No Aplica
- `pdf-transformator` como repo separado: vive acá como responsabilidad núcleo
- Endpoints REST para PDF upload (eso lo maneja `pdf-main`)
- HTTP como transporte principal (Redis Streams es el contrato)
- Todo el stack Python: FastAPI, `redis-py`, Beanie/Motor, `pyproject.toml`