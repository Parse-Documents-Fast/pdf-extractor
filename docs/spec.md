# Spec: PDF Extractor Microservice

**Versión:** 1.0  
**Última actualización:** 2026-10-03

---

## 1. Qué es este servicio

`pdf-extractor` es un **microservicio asíncrono en Go** que extrae estructura textual y tabular de PDFs validados, mapea el contenido a **Markdown canónico**, y publica el resultado para que otros servicios lo persistan.

**Responsabilidad única:** transformar bytes PDF → Markdown estructurado.

**Qué NO hace:**
- Almacenar PDFs o resultados en BD (responsabilidad de `pdf-persistance`, no de este servicio)
- Servir endpoints HTTP para subir o descargar archivos (responsabilidad de `pdf-main`)
- Validar PDFs (eso ya ocurrió antes de llegar acá, responsabilidad de `pdf-validator`)
- Gestionar Redis o MongoDB (usa instancias compartidas gestionadas por `pdf-infra`)
- Balancear carga o servir múltiples réplicas (responsabilidad de Traefik + deploy config, no de este repo)

---

## 2. Entradas

El servicio recibe **mensajes JSON en la stream `queue:extraction` de Redis**, entregados por `pdf-main` tras validar un PDF.

### Formato de entrada (una entrada en la stream)

```json
{
  "pdf_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "filename": "informe.pdf",
  "content_base64": "JVBERi0xLjQKJcOkw7zDtsO4CjIgMCBvYmoKPDwvTGVuZ3RoIDMgMCBSPj4K..."
}
```

**Campos:**
- `pdf_id` (string): ID único del documento, asignado por `pdf-main` al validar. Usado para correlacionar entrada → resultado.
- `filename` (string): nombre original del archivo, solo para logging y debugging.
- `content_base64` (string): PDF completo codificado en base64 (ADR-0002).

**Tamaño máximo:** 20 MiB decodificado (controlado por `pdf-main`, no re-validado acá).

**Fuente:** publicado por `pdf-main` via `XADD queue:extraction` después del flujo de validación (ADR-0004, ADR-0005).

---

## 3. Salidas

El servicio publica **dos streams de Redis**, consumidas por `pdf-main`:

### Stream 3a. `queue:extraction-results` — Éxito o error de la extracción

```json
{
  "pdf_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "markdown_content": "# Título del Documento\n\nPrimer párrafo...\n\n| Encabezado 1 | Encabezado 2 |\n|---|---|\n| Celda 1 | Celda 2 |",
  "status": "success",
  "error": null
}
```

O en caso de error:

```json
{
  "pdf_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "markdown_content": null,
  "status": "failed",
  "error": "ErrNoExtractableText"
}
```

**Campos:**
- `pdf_id`: mismo que la entrada, para correlación.
- `markdown_content`: texto Markdown estructurado si `status == "success"`, null si `"failed"`.
- `status`: literal `"success"` o `"failed"`.
- `error`: cadena identificando el tipo de error (sentinela Go como string, ej. `"ErrCorruptPDF"`, `"ErrNoExtractableText"`), null si éxito.

**Consumidor:** `pdf-main` lee esta stream vía `XREADGROUP queue:extraction-results pdf-main-extractor-consumer` y procesa el resultado (persistir o registrar fallo).

### Stream 3b. `queue:extraction-dlq` — Dead Letter Queue (solo si aplica)

Si un mensaje falla permanentemente (ej. PDF corrupto) **después de N reintentos**, se publica acá para investigación manual:

```json
{
  "pdf_id": "665f1a2b3c4d5e6f7a8b9c0d",
  "original_message": { /* el mensaje de entrada duplicado */ },
  "error": "ErrCorruptPDF",
  "attempt_count": 5,
  "last_attempt_at": "2026-10-03T14:23:45Z"
}
```

**Consumidor:** manual, operaciones o monitoreo. No parte del flujo automático.

---

## 4. Contrato de errores

Todos los errores HTTP que devuelva el servicio (en los endpoints secundarios, ver sección 6) siguen **RFC 9457 (ADR-0001)**, con `Content-Type: application/problem+json`.

Ejemplo:

```json
{
  "type": "about:blank",
  "title": "PDF corrupto",
  "status": 500,
  "detail": "No se pudo parsear el PDF: estructura inválida en el objeto 5.",
  "instance": "/health"
}
```

**Nota:** la mayoría de errores de extracción **no generan respuestas HTTP** — ocurren dentro del consumer loop asíncrono y se publican en `queue:extraction-results` con `status: "failed"`. Solo fallos críticos del servidor (ej. Redis/Mongo caído) generan problemas en HTTP.

---

## 5. Dependencias en otros servicios / ADRs

Este servicio **depende de que exista**:

| Dependencia | Definido en | Qué significa acá |
|---|---|---|
| **Validación previa del PDF** | `pdf-validator` | El PDF ya fue validado (bien formado, tiene texto extraíble). Este servicio asume que entra un PDF válido; si entra corrupto o sin texto, es un caso edge manejado con error. |
| **Formato de error HTTP unificado** | ADR-0001 (RFC 9457) | Todos los errores 5xx que devuelva (raros, en operación) siguen este formato. |
| **DTOs en JSON + base64** | ADR-0002 | La entrada llega en JSON con `content_base64`, este servicio la decodifica y trabaja con bytes. |
| **Transporte Redis Streams + consumer groups** | ADR-0004 | Lee de `queue:extraction` con `XREADGROUP` en un consumer group, publica resultados en `queue:extraction-results`. |
| **Markdown como formato canónico** | ADR-0005 | La salida es Markdown, no HTML. Fusionó las responsabilidades de `pdf-transformator`. |

---

## 6. Interfaz secundaria (HTTP)

El servicio **NO expone endpoints para clientes externos**. Solo dos endpoints internos, consumidos por sistemas de operación y monitoreo:

| Endpoint | Verbo | Qué hace | Consumidor |
|---|---|---|---|
| `/health` | GET | Verifica si el servicio está vivo (liveness). No verifica Redis ni Mongo. | Traefik (health check), scripts de monitoreo. |
| `/metrics` | GET | Expone métricas Prometheus (PDFs procesados, errores, latencia). | Prometheus scraper. |

**Puertos:**
- HTTP (health/metrics): `:8080` (interno a la red de servicios, no expuesto al host).
- Redis/Mongo: configurados por variables de entorno, conectan a las instancias compartidas de `pdf-infra` (no instancias propias).

**No hay:**
- Endpoint para procesar PDFs vía HTTP POST.
- Endpoint de descarga de resultados.
- Panel web, UI, dashboard.
- API REST de dominio.

---

## 7. Repositorio

**Organización:** GitHub de la organización, repo independiente.

**Nombre:** `pdf-extractor`

**Lenguaje:** Go (última versión estable, fijada en `go.mod`).

**Binario único:** `cmd/pdf-extractor/main.go` genera un artefacto llamado `pdf-extractor` (no dos binarios como `api` y `pdf-extractor` separados).

**Red:** se despliega en `fast_pdf_network` (la red interna compartida de `pdf-infra`). No crea redes propias.

**Almacenamiento:** ninguno. Usa la instancia compartida de Redis (`redis://pdf-queue:6379/0`, valor por defecto; configurable) y *no toca* MongoDB en absoluto — los resultados los persiste `pdf-main` llamando a `pdf-persistance`.

---

## 8. Criterios de aceptación

El servicio está listo cuando:

1. **Lee correctamente de `queue:extraction`** sin perder mensajes (consumer group con `XACK` post-proceso).
2. **Extrae estructura (texto + posición + tamaño de fuente) de un PDF** sin aplanarlo prematuramente.
3. **Mapea a Markdown** con heurísticas de altura de fuente para encabezados y detección de tablas por alineación de coordenadas.
4. **Publica resultados en `queue:extraction-results`** en el formato especificado.
5. **Maneja errores comunes** (PDF corrupto, sin texto extraíble, PDF protegido) sin tumbar el consumer, publicando el fallo en el resultado.
6. **Reintentos y DLQ:** reintenta transitorios (Redis caído), escala permanentes a DLQ sin bloquear.
7. **Tests unitarios + integración** con `testcontainers-go` verifican el pipeline end-to-end.
8. **Métricas Prometheus** en `/metrics` (cantidad procesada, errores por tipo, latencia).
9. **/health** responde sin fallar (no verifica dependencias, es liveness solamente).
10. **Graceful shutdown** ante SIGTERM/SIGINT: deja procesar mensajes en vuelo, cierra transacciones y sale limpio.

---

## 9. Lo que NO entra en este servicio

- **Nginx o balanceador de carga:** eso es responsabilidad de Traefik en la capa de orquestación, no de este repo.
- **Redis o MongoDB propios:** usa los compartidos de `pdf-infra`.
- **Persistencia de resultados en BD:** `pdf-main` llama a `pdf-persistance` para eso.
- **Validación de PDF:** ya ocurrió en `pdf-validator`.
- **Conversión Markdown → otros formatos:** eso es responsabilidad de `pdf-converter`.
- **Endpoints REST para clientes:** solo `/health` y `/metrics`.
- **Múltiples binarios en el mismo repo:** un único `main.go`.

---

## 10. Cambios respecto al plan anterior

El plan anterior (`plan-pdf-extractor.md`) contenía desviaciones arquitectónicas que se corrigen acá:

| Desviación | Razón | Corrección |
|---|---|---|
| MongoDB propia (`MONGO_DB=pdf-extractor`) | No es responsabilidad de este servicio | Se elimina: cero persistencia en este repo. |
| Redis propio con `allkeys-lru` | Riesgo de pérdida de jobs | Se elimina: usa la instancia compartida `pdf-queue` con `noeviction`. |
| Servicio `api` con HTTP expuesto, puertos, healthcheck | Confunde responsabilidades | Se elimina: un solo binario `pdf-extractor`, endpoints HTTP secundarios solamente. |
| Red propia `pdf-network` (`172.20.0.0/16`) | Aislamiento innecesario | Se elimina: despliega en `fast_pdf_network` compartida. |
| Dos carpetas de test (`test/`, `tests/`, `testdata/`) | Desorden | Se estandariza: tests a nivel paquete (`*_test.go`), `testdata/` solo para archivos de fixture. |
| Dos binarios (`cmd/api/`, `cmd/pdf-extractor/`) | Confunde propósito | Se unifica: un solo `cmd/pdf-extractor/main.go`. |

---

## 11. Notas técnicas

### Librería PDF
- A elegir en el spike de la fase 2 del plan (candidatos: `pdfcpu`, fork de `rsc.io/pdf`, `go-fitz`).
- Criterio: licencia compatible, acceso a posición + tamaño de fuente, robustez ante PDFs malformados, sin cgo si es posible.
- El spike genera un ADR en `docs/ADR-XXXX-libreria-pdf.md` registrando la decisión.

### Detección de tablas
- No es responsabilidad de la librería (ninguna lo da).
- Heurística propia: agrupar bloques por alineación de coordenadas (Y para filas, X para columnas).
- Iteración con PDFs reales ajusta los umbrales de tolerancia.

### Manejo de errores
- `ExtractStructure()` devuelve `error` tipado (sentinelas con `errors.Is/As`).
- `recover()` alrededor de la librería para capturar panics de archivos malformados.
- Errores permanentes (PDF corrupto, sin texto): publicar en resultado con `status: failed`.
- Errores transitorios (Redis caído): reintento con backoff hasta N intentos, luego DLQ.

### Concurrencia
- Consumer loop: un (1) goroutine por defecto, configurable vía env var si `pdf-main` escala mucho.
- Pool de workers dentro de esa goroutine si la librería PDF es thread-safe (a validar en spike).

### Context y timeouts
- `context.Background()` del consumer loop.
- `context.WithTimeout()` para cada documento (default 30s, configurable).
- Graceful shutdown: `signal.NotifyContext()` en main, propaga a consumers.

---

## 12. Interfaz con `pdf-main`

`pdf-main` es el orquestador que:

1. **Publica** el PDF validado en `queue:extraction`.
2. **Lee resultados** de `queue:extraction-results`.
3. **Llama a `pdf-persistance`** para guardar el Markdown si el resultado fue éxito.
4. **Registra el fallo** en su logging si el resultado fue `failed`.

El contrato entre ambos es:
- **Stream `queue:extraction`:** formato definido en sección 2.
- **Stream `queue:extraction-results`:** formato definido en sección 3.
- **Ninguna otra comunicación:** no hay HTTP directo entre ellos, no hay callbacks, no hay webhooks.

---

## 13. Cambios esperados en otros repos

Para que este servicio funcione, otros repos **ya deben tener hecho**:

- `pdf-main`: publicar en `queue:extraction` tras validar (ya definido en ADR-0004).
- `pdf-persistance`: estar listo para que `pdf-main` le escriba los Markdown (ya definido).
- `pdf-infra`: instancia Redis compartida en `pdf-queue` con `noeviction` + AOF, instancia Mongo compartida.
- `pdf-validator`: validar antes de encolar (responsabilidad previa).

**Este servicio no genera cambios request en otros repos.** Es un consumidor puro.

---

## Resumen: una frase

> `pdf-extractor` es un consumer de Redis Streams que toma PDFs validados, extrae su estructura, mapea a Markdown, y publica el resultado para que `pdf-main` lo persista. Nada de BD propia, nada de HTTP público, nada de balanceo — solo extracción pura y async.