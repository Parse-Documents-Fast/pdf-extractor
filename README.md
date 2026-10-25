# PDF Extractor

Microservicio asíncrono en Go diseñado para extraer Markdown estructurado a partir de documentos PDF, integrado mediante Redis Streams.

## Arquitectura y Especificaciones

- **Stream de Entrada**: `queue:extraction` (Consumer Group: `pdf-extractor-group`, Consumer Name: hostname del contenedor).
- **Stream de Salida**: `queue:extraction-results`
- **Dead Letter Queue (DLQ)**: `queue:extraction-dlq`
- **API HTTP**:
  - `POST /extract`: Endpoint síncrono de extracción de PDF.
  - `GET /health`: Endpoint de verificación de salud.
  - `GET /metrics`: Métricas de Prometheus.

## Variables de Entorno

| Variable | Valor por Defecto | Descripción |
|---|---|---|
| `REDIS_URL` | `redis://localhost:6379/0` | URI de conexión a Redis |
| `REDIS_STREAM_INPUT` | `queue:extraction` | Nombre del stream de entrada |
| `REDIS_STREAM_OUTPUT` | `queue:extraction-results` | Nombre del stream de salida |
| `REDIS_STREAM_DLQ` | `queue:extraction-dlq` | Nombre del stream DLQ |
| `REDIS_CONSUMER_GROUP` | `pdf-extractor-group` | Nombre del consumer group |
| `CONSUMER_NAME` | Hostname del contenedor | Identificador único de consumer por réplica |
| `HTTP_ADDR` | `:8000` | Dirección del servidor HTTP |
| `LOG_LEVEL` | `info` | Nivel de log (`debug`, `info`, `warn`, `error`) |
| `WORKER_COUNT` | `1` | Número de workers locales de extracción |
| `MAX_FILE_SIZE_MB` | `20` | Tamaño máximo permitido de PDF en MB |

## Logging y Privacidad

El servicio emite logs estructurados en formato JSON hacia `stdout` utilizando `log/slog`.
- **Niveles**: `debug`, `info`, `warn`, `error`.
- **Política de Privacidad**: Está estrictamente prohibido registrar los bytes del PDF, el texto extraído ni el contenido del documento (incluso en nivel `debug`). Únicamente se registran metadatos operativos (tamaño, checksum, ID, conteo de páginas, duración y códigos de error).

## Puesta en Marcha y Ejecución

### 1. Configuración del Entorno
```bash
cp .env.example .env
```

### 2. Ejecución con Docker
```bash
docker compose up --build -d
```
Para escalar horizontalmente (ej. 5 réplicas):
```bash
PDF_EXTRACTOR_REPLICAS=5 docker compose up --build -d
```

### 3. Desarrollo Local (Makefile)
```bash
make build   # Compila el binario
make test    # Ejecuta tests unitarios e integración
make run     # Ejecuta el servicio
```

## Documentación
Consulte [commands.md](commands.md) para comandos de operación y [docs/spec.md](docs/spec.md) para especificaciones técnicas detalladas.
