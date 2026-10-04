# Plan de Desarrollo — PDF Extractor Microservice (Go)

**Metodología:** SDD (Spec-Driven Development), ADR-0003  
**Spec de referencia:** `docs/spec.md` (este repo)  
**Última actualización:** 2026-10-03

---

## Resumen Ejecutivo

Microservicio consumer de Redis Streams que extrae estructura de PDFs y mapea a Markdown. Cero persistencia, cero HTTP público, cero balanceador propio. Lee de `queue:extraction`, publica resultados en `queue:extraction-results`.

**Stack:** Go, `github.com/redis/go-redis/v9`, ninguna BD propia.

**Salida:** un binario `pdf-extractor`, desplegable en cualquier cantidad de replicas vía orquestador (Traefik, Kubernetes, etc.).

---

## Estructura del Repositorio

```
pdf-extractor/
├── cmd/pdf-extractor/
│   └── main.go               # Único entry point: config, wiring, consumer loop, graceful shutdown
├── internal/
│   ├── config/
│   │   └── config.go         # Carga env vars, valida, devuelve Config{} tipado
│   ├── extractor/
│   │   ├── extractor.go      # Block, ExtractStructure, PdfExtractionError
│   │   └── extractor_test.go
│   ├── markdown/
│   │   ├── markdown.go       # MapStructureToMarkdown
│   │   └── markdown_test.go
│   ├── consumer/
│   │   ├── consumer.go       # Consumer loop, XREADGROUP, XACK, reintentos
│   │   └── consumer_test.go
│   └── models/
│       └── models.go         # Message structs (JSON tags snake_case)
├── test/
│   └── integration_test.go   # Tests de integración con Redis + testcontainers
├── testdata/
│   ├── simple.pdf            # PDF sin tablas
│   ├── table.pdf             # PDF con tabla
│   ├── corrupted.pdf         # PDF malformado
│   ├── no-text.pdf           # PDF escaneado (sin texto extraíble)
│   ├── simple.pdf.md         # Expected output para simple.pdf (golden file)
│   └── table.pdf.md
├── docs/
│   ├── spec.md               # [copia de spec.md desde este mismo plan]
│   └── ADR-XXXX-*.md         # Decision records (spike libr. PDF, etc.)
├── Dockerfile                # Multi-stage, imagen mínima
├── docker-compose.test.yml   # Para tests locales (Redis, Mongo si aplica — pero acá no)
├── Makefile                  # Targets: build, test, lint, run, clean
├── go.mod                    # go.sum, dependencias pinned
├── .env.example              # Variables de entorno con defaults
├── .golangci.yml             # Config de linter
├── README.md                 # Setup, cómo correr, cómo testear, contrato de streams
└── .gitignore
```

**No hay:**
- `cmd/api/main.go` (bifurcación descartada)
- `internal/nginx.conf` (Traefik lo maneja)
- `internal/config/nginx.go`
- Múltiples carpetas de test (`tests/`, `test/`, `testdata/` mezcladas)
- `docker-compose.tp.with-lb.yml` con 5 réplicas manuales (eso se configura en el orquestador)
- MongoDB en este repo

---

## Variables de Entorno

```bash
# Redis (consumer loop)
REDIS_URL=redis://localhost:6379/0            # Stream de entrada/salida
REDIS_CONSUMER_GROUP=pdf-extractor-group      # Nombre del consumer group
REDIS_STREAM_INPUT=queue:extraction
REDIS_STREAM_OUTPUT=queue:extraction-results
REDIS_STREAM_DLQ=queue:extraction-dlq
REDIS_MAX_RETRIES=5                           # Reintentos antes de DLQ
REDIS_RETRY_BACKOFF_MS=1000                   # Backoff inicial (exponencial)

# HTTP (health/metrics)
HTTP_ADDR=:8080
HTTP_SHUTDOWN_TIMEOUT=15s

# Procesamiento
PDF_EXTRACTION_TIMEOUT=30s                    # Timeout por documento
PDF_MAX_BYTES=20971520                        # 20 MiB (validado por pdf-main, pero validamos de nuevo)
FONT_SIZE_HEADING_THRESHOLD=1.5               # Múltiplo del tamaño predominante
FONT_SIZE_SUBHEADING_THRESHOLD=1.2
TABLE_CELL_X_TOLERANCE_PT=5                   # Tolerancia en puntos para alinear columnas
TABLE_CELL_Y_TOLERANCE_PT=3

# Logging
LOG_LEVEL=info                                # debug, info, warn, error
LOG_FORMAT=json                               # json, text

# Concurrencia
WORKER_COUNT=1                                # Workers paralelos (default 1; > 1 solo si librería PDF es thread-safe)
```

**Archivo `.env.example`:** incluir todos con valores por defecto razonables.

---

## Fases

### ⚠️ Fase 0: Spike — Elegir librería PDF

**Artefacto:** ADR en `docs/ADR-XXXX-libreria-pdf.md`

Candidatos:

| Librería | Licencia | Posición + fuente | Robustez | Sin CGO | Cons |
|---|---|---|---|---|---|
| `unipdf` | AGPL/Comercial | ✅ | ✅ | ✅ | **API key obligatoria**, validar con legal antes |
| `pdfcpu` | Apache 2.0 | ❌ No | ✅ | ✅ | Pensada para manipular, no extraer con posición |
| `rsc.io/pdf` (forks) | BSD | ✅ | ⚠️ Media | ✅ | No activamente mantenido, pero stable |
| `go-fitz` (MuPDF) | AGPL | ✅ | ✅ | ❌ CGO | MuPDF es AGPL, impacto en build/Docker |
| `go-pdfium` | Apache 2.0 | ✅ | ✅ | ⚠️ Wasm/CGO | Nuevo, menos maduro, más pesado |

**Tareas:**

- [ ] Prototipo funcional por librería: decodificar PDF, extraer bloques con posición, loguear resultado.
- [ ] Test con los 4 PDFs en `testdata/` (simple, tabla, corrupto, sin texto).
- [ ] Tabla comparativa (tiempo de build, tamaño de imagen Docker, memoria en runtime, precisión de coordenadas).
- [ ] Decision: elegir una y registrar en ADR.

**Entrada a Fase 1 si:** se eligió una librería y hay evidencia de que funciona con los PDFs de test.

---

### Fase 1: Estructura Base + Consumer Loop (stub)

**Punto de control:** servicio arranca, se une a consumer group, consume un mensaje de test, se apaga limpio sin perder el mensaje.

#### 1.1. Setup inicial

- [ ] `go mod init github.com/<org>/pdf-extractor`
- [ ] `Makefile` con targets:
  - `make build` → compila `cmd/pdf-extractor/main.go` a `./bin/pdf-extractor`
  - `make test` → corre todos los tests
  - `make lint` → `golangci-lint run`
  - `make run` → ejecuta con `.env.local` (si existe, else `.env.example`)
  - `make clean` → limpia `./bin`
- [ ] `.golangci.yml` estándar para Go (errcheck, govet, golint mínimo)
- [ ] `.gitignore`: `bin/`, `.env.local`, IDE noise
- [ ] `README.md` inicial: descripción, stack, setup, cómo correr, cómo testear

#### 1.2. Config tipado + logging

- [ ] `internal/config/config.go`:
  - Struct `Config` con todos los campos env var (Redis URL, timeouts, etc.)
  - Función `Load()` que lee `.env`, usa defaults, valida
  - Falla en startup si algún campo obligatorio falta (no defaults silenciosos)
- [ ] `log/slog` (estándar en Go 1.21+), en JSON por defecto
  - Wrapper `internal/logger/logger.go` con helpers (`Info`, `Error`, `Debugf`, etc.)
  - Inject en los componentes vía parámetro (sin var global)

#### 1.3. Cliente Redis + consumer group

- [ ] `internal/redis/client.go`:
  - Inicializar `*redis.Client` desde `Config.RedisURL`
  - Health check en startup (ping, si falla: log y continuar — no es bloqueante)
  - Defer `.Close()` en shutdown
- [ ] Crear consumer group en startup:
  ```go
  XGROUP CREATE <stream> <group> $ MKSTREAM
  ```
  Tolerara `BUSYGROUP` (ya existe).
- [ ] Clase `RedisConsumer`:
  - Método `ReadMessages(ctx)` que llamea `XREADGROUP BLOCK <timeout> STREAMS <stream> >`
  - Devuelve slice de `XMessage`

#### 1.4. Consumer loop (stub)

- [ ] `internal/consumer/consumer.go`:
  - Struct `Consumer` que recibe `RedisClient`, `Logger`, `Config`
  - Método `Start(ctx context.Context) error`
  - Loop que:
    1. `XREADGROUP` desde `REDIS_STREAM_INPUT`
    2. Por cada mensaje:
       - Log el ID y pdf_id (no procesa aún, es stub)
       - `XACK` inmediatamente (marcar como procesado)
       - No hace nada con el contenido
    3. Si `ctx` se cancela, sale del loop
  - No hay reintentos ni DLQ aún (se agregan en fase 5)

#### 1.5. Main + graceful shutdown

- [ ] `cmd/pdf-extractor/main.go`:
  - `main()` que:
    1. Carga config
    2. Crea logger
    3. Conecta Redis
    4. Crea consumer
    5. `signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)`
    6. Arranca consumer en goroutine separada
    7. Espera `<-ctx.Done()`
    8. Cierra cliente Redis, logger (si aplica)
    9. Exit 0
  - En caso de error: log fatal, exit 1

#### 1.6. Dockerfile (multi-stage)

```dockerfile
# Stage 1: build
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o pdf-extractor ./cmd/pdf-extractor

# Stage 2: runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/pdf-extractor /usr/local/bin/
EXPOSE 8080
ENTRYPOINT ["pdf-extractor"]
```

---

### Fase 2: Extracción Estructurada de PDF

**Punto de control:** `ExtractStructure()` extrae bloques de un PDF con posición, tamaño de fuente, tipo (texto o tabla); respeta context timeout; atrapa panics.

#### 2.1. Definir tipos de dominio

- [ ] `internal/extractor/types.go`:
  ```go
  type Block struct {
    Text         string
    FontSize     float64
    X, Y         float64
    Width, Height float64
    Page         int
    IsTableCell  bool
    TableRow, TableCol int  // Si IsTableCell == true
  }

  type PdfExtractionError struct {
    Err  error
    Code string  // "ErrCorruptPDF", "ErrNoExtractableText", etc.
  }
  ```
- [ ] Sentinelas (implementar `Is()` + `Unwrap()`):
  ```go
  var (
    ErrCorruptPDF = errors.New("PDF corrupto")
    ErrNoExtractableText = errors.New("Sin texto extraíble")
    ErrProtectedPDF = errors.New("PDF protegido")
    ErrUnsupportedFormat = errors.New("Formato no soportado")
    ErrTimeout = errors.New("Timeout en extracción")
  )
  ```

#### 2.2. Integrar librería PDF elegida (fase 0 spike)

- [ ] `internal/extractor/extractor.go`:
  - Función `ExtractStructure(ctx context.Context, pdf []byte) ([]Block, error)`
  - Usa la librería elegida en spike
  - Itera PDF página por página
  - Por cada bloque de texto, captura: posición (X, Y), tamaño de fuente, contenido
  - Devuelve slice de `Block` ordenados por página y posición (top-left a bottom-right)
- [ ] Manejo de errores:
  - `defer` un `recover()` alrededor de la librería (algunos PDFs la hacen panic)
  - Wrappear panics como `ErrCorruptPDF`
  - Devolver `PdfExtractionError` tipado con `.Unwrap()`
- [ ] Context timeout:
  - Respetar `ctx.Deadline()` dentro del loop de extracción
  - Si se excede, retornar `ErrTimeout`
- [ ] Golden files en `testdata/`:
  ```
  testdata/
  ├── simple.pdf
  ├── simple.pdf.blocks.json  (JSON con los bloques extraídos esperados)
  ├── table.pdf
  ├── table.pdf.blocks.json
  ├── corrupted.pdf
  └── no-text.pdf
  ```

#### 2.3. Tests unitarios

- [ ] `internal/extractor/extractor_test.go` (table-driven):
  ```go
  func TestExtractStructure(t *testing.T) {
    tests := []struct {
      name    string
      pdfPath string
      want    int        // cantidad de bloques esperados
      wantErr error
    }{
      {"simple", "../../testdata/simple.pdf", 10, nil},
      {"table", "../../testdata/table.pdf", 25, nil},
      {"corrupted", "../../testdata/corrupted.pdf", 0, ErrCorruptPDF},
      {"no-text", "../../testdata/no-text.pdf", 0, ErrNoExtractableText},
    }
    for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) {
        got, err := ExtractStructure(context.Background(), loadTestPDF(tt.pdfPath))
        if tt.wantErr != nil {
          assert.ErrorIs(t, err, tt.wantErr)
        } else {
          assert.NoError(t, err)
          assert.Equal(t, tt.want, len(got))
        }
      })
    }
  }
  ```
- [ ] Benchmark:
  ```go
  func BenchmarkExtractStructure(b *testing.B) {
    pdf := loadTestPDF("../../testdata/simple.pdf")
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
      ExtractStructure(context.Background(), pdf)
    }
  }
  ```

---

### Fase 3: Mapeo a Markdown

**Punto de control:** `MapStructureToMarkdown()` convierte bloques a Markdown con encabezados detectados por tamaño de fuente y tablas detectadas por alineación de coordenadas.

#### 3.1. Heurísticas de mapeo

- [ ] `internal/markdown/markdown.go`:
  - Función `MapStructureToMarkdown(blocks []Block, config MarkdownConfig) string`
  - **Detección de encabezados:**
    - Calcular tamaño de fuente predominante en el doc (moda).
    - `fontsize > predominante * 1.5` → `#` (h1)
    - `fontsize > predominante * 1.2` → `##` (h2)
    - `fontsize > predominante` → `###` (h3)
    - Resto → párrafo plano
  - **Detección de tablas:**
    - Agrupar bloques por `TableRow` / `TableCol` (si vienen con eso de la librería PDF)
    - Si no: agrupar por alineación (Y ±tolerancia → fila, X ±tolerancia → columna)
    - Generar sintaxis Markdown: `| col1 | col2 |` + fila separadora `|---|---|`
  - **Escapado:**
    - Escapar `|` dentro de celdas (reemplazar por `\|` o espacio)
    - Escapar `\`, backticks, `_` en contenido plano donde aplique
  - **Limpieza:**
    - Trimear espacios en blanco entre bloques
    - Colapsar múltiples saltos de línea en máximo 2

#### 3.2. Config del mapeo

- [ ] Struct `MarkdownConfig`:
  ```go
  type MarkdownConfig struct {
    HeadingThreshold      float64  // Múltiplo del predominante (default 1.5)
    SubheadingThreshold   float64  // (default 1.2)
    TableXTolerance       float64  // Puntos (default 5)
    TableYTolerance       float64  // Puntos (default 3)
  }
  ```
- [ ] Integrar en `Config` global desde env vars

#### 3.3. Tests unitarios + golden files

- [ ] `internal/markdown/markdown_test.go`:
  - Para cada PDF en `testdata/`, verificar que el Markdown generado coincida con `*.md` esperado
  - Usar `testify/assert.Equal()` con texto multiline (fácil de debuggear)
  - Casos:
    - Simple sin tablas → párrafos + encabezados por tamaño
    - Con tabla → tabla Markdown formateada
    - Vacío → string vacío, sin error
- [ ] Golden files:
  ```
  testdata/
  ├── simple.pdf.md     # Expected output
  ├── table.pdf.md
  └── ...
  ```
- [ ] Benchmarks de mapeo (generalmente rápido, pero medir)

#### 3.4. Integración en consumer (aún stub)

- [ ] Consumer stub ahora llama internamente:
  ```go
  blocks, err := extractor.ExtractStructure(ctx, pdfBytes)
  if err != nil {
    log.Error("extraction failed", "error", err)
    // Publicar fallo (aún no, eso es Fase 5)
    return
  }
  md := markdown.MapStructureToMarkdown(blocks, config)
  log.Info("extracted markdown", "lines", len(strings.Split(md, "\n")))
  ```

---

### Fase 4: Modelos de mensaje (DTOs)

**Punto de control:** Structs JSON con tags `snake_case` para entrada/salida, decodificación/codificación sin error.

#### 4.1. Tipos de mensaje

- [ ] `internal/models/models.go`:
  ```go
  // Entrada (desde pdf-main)
  type ExtractionRequest struct {
    PdfID       string `json:"pdf_id"`
    Filename    string `json:"filename"`
    ContentB64  string `json:"content_base64"`
  }

  // Salida (hacia pdf-main)
  type ExtractionResult struct {
    PdfID            string `json:"pdf_id"`
    MarkdownContent  *string `json:"markdown_content"` // nil si failed
    Status           string  `json:"status"`  // "success" o "failed"
    Error            *string `json:"error"`   // nil si success
  }

  // DLQ (investigación manual)
  type DlqMessage struct {
    PdfID            string            `json:"pdf_id"`
    OriginalMessage  ExtractionRequest `json:"original_message"`
    Error            string            `json:"error"`
    AttemptCount     int               `json:"attempt_count"`
    LastAttemptAt    time.Time         `json:"last_attempt_at"`
  }
  ```

#### 4.2. Parseo y validación

- [ ] Función `ParseExtractionRequest(rawMsg string) (*ExtractionRequest, error)`:
  - Decodificar JSON
  - Validar campos obligatorios (`pdf_id`, `filename`, `content_base64`)
  - Decodificar base64 de `content_base64` (devolver bytes, no guardar el string)
  - Verificar tamaño del PDF decodificado (< `PDF_MAX_BYTES`)
  - Devolver error tipado si falla
- [ ] Función `EncodeExtractionResult(result *ExtractionResult) (string, error)`:
  - Codificar a JSON
  - Verificar que sea válido (no contiene NaN, Inf, etc.)

#### 4.3. Tests

- [ ] Casos de parsing:
  - Mensaje válido → parsea correctamente
  - Campo faltante (`pdf_id`) → error de validación
  - `content_base64` malformado (no es base64 válido) → error
  - PDF decodificado > máximo → error de tamaño

---

### Fase 5: Integración completa con Redis Streams + reintentos + DLQ

**Punto de control:** Consumer procesa entrada → extrae → mapea → publica resultado; reintenta en transitorios; escala a DLQ en permanentes.

#### 5.1. Flujo real del consumer

- [ ] `internal/consumer/consumer.go` reemplaza stub:
  1. `XREADGROUP` desde `REDIS_STREAM_INPUT`
  2. Por cada mensaje:
     - Parsear JSON → `ExtractionRequest`
     - Decodificar base64
     - Llamar `ExtractStructure(ctx, pdfBytes)`
     - Si error permanente → publicar en resultado con `status: failed`
     - Si error transitorio → reintento (ver 5.2)
     - Si éxito → `MapStructureToMarkdown` → publicar en resultado con `status: success`
     - `XACK` solo DESPUÉS de publicar resultado
  3. Loop continúa

#### 5.2. Reintentos con backoff exponencial

- [ ] Struct `RetryPolicy`:
  ```go
  type RetryPolicy struct {
    MaxAttempts      int           // default 5
    InitialBackoff   time.Duration // default 1s
    MaxBackoff       time.Duration // default 30s
  }
  ```
- [ ] Función `ShouldRetry(err error) bool`:
  - `true` si es error transitorio (Redis error, context timeout, etc.)
  - `false` si es error permanente (PDF corrupto, validación, etc.)
- [ ] Función `NextBackoff(attempt int, policy RetryPolicy) time.Duration`:
  - `backoff = initial * 2^attempt`
  - Cap a `MaxBackoff`
- [ ] En el consumer loop:
  ```go
  attempt := 1
  for {
    err := process(msg)
    if err == nil {
      XACK(msg)
      break
    }
    if !ShouldRetry(err) || attempt >= policy.MaxAttempts {
      publishResult(msg, failed, err)
      XACK(msg)
      break
    }
    time.Sleep(NextBackoff(attempt, policy))
    attempt++
  }
  ```

#### 5.3. Dead Letter Queue (DLQ)

- [ ] Si se exceden intentos:
  - Publicar en `REDIS_STREAM_DLQ` con estructura `DlqMessage`
  - Log con nivel WARN o ERROR
  - NO reintenta más
  - `XACK` el original para no quedar atascado
- [ ] Consumer de DLQ (manual):
  - No hay consumer automático en este servicio
  - Operaciones leen manualmente para investigar
  - Formato JSON permite reintento manual si se decide

#### 5.4. Concurrencia configurable

- [ ] Env var `WORKER_COUNT` (default 1)
- [ ] En main: lanzar N goroutines, cada una corre el consumer loop
- [ ] Importante: **validar que la librería PDF es thread-safe** (agregarse al spike de Fase 0 si no se hizo)
- [ ] Si no es thread-safe, mantener `WORKER_COUNT=1` y documentar

#### 5.5. Tests de integración

- [ ] `test/integration_test.go` con `testcontainers-go`:
  ```go
  func TestConsumerIntegration(t *testing.T) {
    // Spin up Redis container
    // Start consumer goroutine
    // Publish a valid message
    // Assert result appears in output stream within timeout
    // Verify XACK was called (message gone from pending)
  }

  func TestConsumerRetry(t *testing.T) {
    // Publish message that will fail once, then succeed
    // Assert it's retried and eventually succeeds
  }

  func TestConsumerDLQ(t *testing.T) {
    // Publish message that always fails
    // Assert it goes to DLQ after max retries
  }
  ```

---

### Fase 6: Endpoints HTTP Secundarios

**Punto de control:** `/health` responde 200 OK, `/metrics` expone contadores Prometheus.

#### 6.1. Health check

- [ ] `internal/httpserver/health.go`:
  - Endpoint `GET /health`
  - Devuelve 200 OK siempre (liveness check, no verifica dependencias)
  - Body: `{"status":"ok"}`
- [ ] NO hacer `PING` a Redis o Mongo en `/health` (eso lentificaría el check)
- [ ] La cátedra o Traefik puede usar otro endpoint si quiere readiness (p. ej. `/ready`) que sí verifique deps

#### 6.2. Métricas Prometheus

- [ ] `internal/httpserver/metrics.go`:
  - Endpoint `GET /metrics`
  - Expone:
    - `pdf_extractor_processed_total{status="success|failed"}` (Counter)
    - `pdf_extractor_errors_total{error_type="ErrCorruptPDF|ErrNoExtractableText|..."}` (Counter)
    - `pdf_extractor_latency_seconds{quantile="0.5|0.95|0.99"}` (Histogram)
    - `pdf_extractor_dlq_messages_total` (Counter)
    - `pdf_extractor_pending_messages` (Gauge, cantidad en Redis Streams `XPENDING`)
- [ ] Registrar métricas en el consumer loop (thread-safe con `sync.Mutex` o channel si es necesario)

#### 6.3. Servidor HTTP

- [ ] `internal/httpserver/server.go`:
  - Crear mux con ambos endpoints
  - Config de `ReadTimeout`, `WriteTimeout` desde `Config`
  - Graceful shutdown: `server.Shutdown(ctx)` con timeout
- [ ] Main: arranca servidor en goroutine separada, escucha en `HTTP_ADDR`

---

### Fase 7: Tests Unitarios Completos + Cobertura

**Punto de control:** `go test ./...` pasa todo, cobertura > 80%, CI green.

#### 7.1. Tests por paquete

| Paquete | Tests |
|---|---|
| `config` | Carga env vars, defaults, validación fallida |
| `extractor` | ExtractStructure con PDFs test, error handling |
| `markdown` | MapStructureToMarkdown con golden files |
| `consumer` | Parseo de mensaje, reintentos, DLQ (unitarios, mocks) |
| `models` | Serialización/deserialización JSON, edge cases |
| `httpserver` | Health check, metrics format (sin levantar Redis) |
| `integration` | Consumer + Redis + testcontainers (Fase 5) |

#### 7.2. Coverage

- [ ] `make test COVERAGE=true` genera reporte HTML
- [ ] Meta: > 80% overall
- [ ] Excluir: main.go (wiring), testdata/ (fixtures)

#### 7.3. Lint

- [ ] `golangci-lint run` limpio
- [ ] Config `.golangci.yml` con rules sensatos (no extremistas)

#### 7.4. CI pipeline

- [ ] `.github/workflows/test.yml`:
  ```yaml
  on: [push, pull_request]
  jobs:
    test:
      runs-on: ubuntu-latest
      steps:
        - uses: actions/checkout@v3
        - uses: actions/setup-go@v4
          with:
            go-version: '1.22'
        - run: go mod download
        - run: make lint
        - run: make test COVERAGE=true
        - uses: codecov/codecov-action@v3
          with:
            files: ./coverage.out
  ```

---

### Fase 8: Documentación Final

**Punto de control:** README + docs/ completos, cualquiera del equipo puede levantar el servicio.

#### 8.1. README.md

- [ ] Descripción de una línea
- [ ] Stack (Go, Redis, librería PDF)
- [ ] Requisitos (Go 1.22+, Docker, Redis local para tests)
- [ ] Quickstart:
  ```bash
  cp .env.example .env.local
  make build
  make run
  ```
- [ ] Cómo testear:
  ```bash
  make test
  docker-compose -f docker-compose.test.yml up -d
  make test  # Ahora con Redis real
  ```
- [ ] Contrato de Redis Streams (formato de entrada/salida)
- [ ] Cómo deployar (imagen Docker, env vars, orquestador)

#### 8.2. Docs en `docs/`

- [ ] `docs/spec.md`: copia de este plan (o referencia a doc compartido)
- [ ] `docs/ADR-XXXX-libreria-pdf.md`: decisión de la Fase 0
- [ ] `docs/architecture.md`: diagrama ASCII del consumer loop (opcional pero nice to have)

#### 8.3. Ejemplos

- [ ] `docs/examples/` (opcional):
  - Script de test: cómo publicar un mensaje en `queue:extraction` manualmente y ver el resultado en `queue:extraction-results`
  - Docker Compose para levantar Redis + servicio localmente

---



## Criterios de Definición de Hecho

**Por Fase:**

| Fase | DoD |
|---|---|
| 0 | ADR decidido y registrado, spike code descartado/archivado |
| 1 | Binario `pdf-extractor` arranca, consume de Redis, se apaga limpio |
| 2 | `ExtractStructure()` extrae bloques con posición + tamaño de fuente; PDFs test pasan |
| 3 | `MapStructureToMarkdown()` mapea bloques a Markdown; golden files coinciden |
| 4 | JSON parsea y serializa correctamente con tags `snake_case` |
| 5 | Consumer loop completo, reintenta, publica resultado, XACK; DLQ funciona |
| 6 | `/health` 200, `/metrics` expone contadores Prometheus válidos |
| 7 | `go test ./...` pasa, coverage > 80%, CI verde, lint limpio |
| 8 | README claro, docs actualizadas, equipo puede levantar y deployar |

**Final:** Spec match 100%, todo bajo control de versión, CI/CD integrado, listo para merge a main.

---

## Puntos de Control Críticos

1. **Librería PDF elegida (Fase 0):** si elige una mala (sin posición, lenta, licencia problemática), todo se ralentiza o muere.
2. **Consumer loop correcto (Fase 5):** si falla la integración con Redis Streams, reintentos o XACK, los mensajes se pierden o quedan atascados.
3. **Graceful shutdown (Fase 1, validar en Fase 5):** si no desliga bien, en deploy pueden perder mensajes en vuelo.

---

## Notas Finales

- **No hay MongoDB en este repo.** Los resultados los persiste `pdf-main` llamando a `pdf-persistance`.
- **Un binario, no dos.** No existe `cmd/api/main.go`.
- **Traefik maneja réplicas.** No reinventar con nginx o docker-compose.
- **Redis compartida.** La misma instancia que usa todo el sistema, configurada con `noeviction + AOF`.
- **Spec first, luego código.** Este plan es la fuente de verdad; cualquier desviación es un cambio de scope que requiere actualizar spec + plan.