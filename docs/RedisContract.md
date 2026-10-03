# Redis Stream Contract

## Visión General

PDF Extractor usa Redis Streams para comunicación asíncrona entre:
- **pdf-main** → Productor (envía PDFs)
- **pdf-extractor** → Consumidor (procesa PDFs)

Dos streams: `queue:extraction` (entrada) y `queue:extraction-results` (salida)

---

## Stream de Entrada: `queue:extraction`

### Estructura del Mensaje

```json
{
  "pdf_id": "string",
  "filename": "string",
  "content_base64": "string"
}
```

### Campos Detallados

| Campo | Tipo | Descripción | Ejemplo |
|-------|------|-------------|---------|
| `pdf_id` | string | ID único del PDF | `"pdf_12345abc"` |
| `filename` | string | Nombre original del archivo | `"documento.pdf"` |
| `content_base64` | string | PDF codificado en base64 | `"JVBERi0xLjQK..."` |

### Explicación de content_base64

El contenido del PDF se codifica en base64 porque:
- Redis Streams maneja texto UTF-8
- Base64 permite transportar datos binarios como texto
- Tamaño máximo recomendado: 20MB (configurable en `PDF_MAX_BYTES`)

**Ejemplo (PDF pequeño):**
```
Archivo: documento.pdf (50 bytes)
↓ (codificar en base64)
content_base64: "JVBERi0xLjQKCjEgMCBvYmo..."
```

### Comando: XADD (Agregar mensaje)

**Desde pdf-main:**

```bash
redis-cli XADD queue:extraction "*" \
  pdf_id "pdf_67890xyz" \
  filename "factura_2024.pdf" \
  content_base64 "JVBERi0xLjQKCjEgMCBvYmogICUgZW50cnkg..."
```

**Respuesta:**
```
"1727900400000-0"
```

El ID de stream (timestamp-sequence) se usa internamente.

### Ejemplo Real (Python/GO)

**Python (pdf-main):**
```python
import redis
import base64

r = redis.Redis(host='redis', port=6379, decode_responses=True)

with open('factura.pdf', 'rb') as f:
    pdf_content = f.read()
    content_b64 = base64.b64encode(pdf_content).decode('utf-8')

r.xadd('queue:extraction', {
    'pdf_id': 'pdf_20241003_001',
    'filename': 'factura.pdf',
    'content_base64': content_b64
})
```

**Go (pdf-main):**
```go
package main

import (
	"context"
	"encoding/base64"
	"io/ioutil"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	
	pdfBytes, _ := ioutil.ReadFile("factura.pdf")
	contentB64 := base64.StdEncoding.EncodeToString(pdfBytes)
	
	rdb.XAdd(context.Background(), "queue:extraction", map[string]interface{}{
		"pdf_id":          "pdf_20241003_001",
		"filename":        "factura.pdf",
		"content_base64":  contentB64,
	})
}
```

---

## Stream de Salida: `queue:extraction-results`

### Estructura del Mensaje

```json
{
  "pdf_id": "string",
  "markdown_content": "string",
  "status": "string",
  "error": "string"
}
```

### Campos Detallados

| Campo | Tipo | Descripción | Valores/Ejemplo |
|-------|------|-------------|-----------------|
| `pdf_id` | string | ID del PDF procesado | `"pdf_67890xyz"` |
| `markdown_content` | string | Contenido extraído en Markdown | `"# Factura\n\n..."` |
| `status` | enum | Estado del procesamiento | `"success"` o `"failed"` |
| `error` | string | Mensaje de error (si aplica) | `null` o `"timeout"` |

### Status Posibles

| Status | Descripción | error | Acción |
|--------|-------------|-------|--------|
| `success` | Extracción exitosa | null | Guardar `markdown_content` |
| `failed` | Error en extracción | texto | Registrar error, reintentar |

### Comando: XREAD (Leer resultados)

**Desde pdf-main (leer respuestas):**

```bash
redis-cli XREAD COUNT 10 STREAMS queue:extraction-results 0
```

**Respuesta:**
```
1) 1) "queue:extraction-results"
   2) 1) 1) "1727900410000-0"
         2) 1) "pdf_id"
            2) "pdf_67890xyz"
            3) "markdown_content"
            4) "# Factura\n\nNúmero: 001-2024\n..."
            5) "status"
            6) "success"
            7) "error"
            8) ""
```

### Ejemplo Real (Python/GO)

**Python (pdf-main - leer resultados):**
```python
import redis

r = redis.Redis(host='redis', port=6379, decode_responses=True)

# Leer últimos 10 resultados
results = r.xread({'queue:extraction-results': '0'}, count=10)

for stream_name, messages in results:
    for msg_id, msg_data in messages:
        pdf_id = msg_data['pdf_id']
        status = msg_data['status']
        markdown = msg_data['markdown_content']
        error = msg_data.get('error', '')
        
        if status == 'success':
            print(f"PDF {pdf_id}: Extracción exitosa")
            # Guardar markdown a BD
        else:
            print(f"PDF {pdf_id}: Error - {error}")
```

**Go (pdf-main - leer resultados):**
```go
package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: "redis:6379"})
	
	// Leer últimos 100 resultados
	results := rdb.XRead(context.Background(), &redis.XReadArgs{
		Streams: []string{"queue:extraction-results", "0"},
		Count:   100,
	})
	
	for _, msg := range results.Val()[0].Messages {
		pdfID := msg.Values["pdf_id"].(string)
		status := msg.Values["status"].(string)
		markdown := msg.Values["markdown_content"].(string)
		
		if status == "success" {
			fmt.Printf("PDF %s: Success\n", pdfID)
		}
	}
}
```

---

## Flujo Completo

### Paso 1: pdf-main envía PDF

```
pdf-main
  └─ Lee archivo: factura.pdf
  └─ Codifica base64
  └─ XADD queue:extraction
     {
       pdf_id: "pdf_20241003_001",
       filename: "factura.pdf",
       content_base64: "JVBERi0xLjQK..."
     }
```

### Paso 2: pdf-extractor procesa

```
pdf-extractor (consumer)
  └─ XREAD queue:extraction (bloquea esperando)
  └─ Recibe mensaje
  └─ Decodifica base64
  └─ Extrae contenido → Markdown
  └─ Almacena en MongoDB
  └─ XADD queue:extraction-results
     {
       pdf_id: "pdf_20241003_001",
       markdown_content: "# Factura\n\nNúmero...",
       status: "success"
     }
```

### Paso 3: pdf-main lee resultados

```
pdf-main
  └─ XREAD queue:extraction-results (con polling)
  └─ Recibe:
     {
       pdf_id: "pdf_20241003_001",
       markdown_content: "# Factura...",
       status: "success"
     }
  └─ Actualiza BD local
  └─ Notifica usuario
```

---

## Punto de Coordinación con pdf-main

### Identificación: pdf_id

El `pdf_id` es la **llave de coordinación**:
- pdf-main crea un pdf_id único
- pdf-main envía en `queue:extraction`
- pdf-extractor devuelve el mismo `pdf_id`
- pdf-main matchea entrada/salida por `pdf_id`

**Ejemplo de flujo coordinado:**

```
Tiempo  | pdf-main          | Redis                    | pdf-extractor
--------|-------------------|--------------------------|---------------
T0      | pdf_id=ABC        |                          |
T1      | XADD entrada      | {ABC, factura.pdf, ...}  |
T2      |                   | Consumer pickup          | Procesa ABC
T3      |                   |                          | XADD salida
T4      | XREAD salida      | {ABC, markdown, ...}     |
T5      | Actualiza BD      |                          |
T6      | Marca como done   |                          |
```

### Vinculación con Base de Datos

```sql
-- Tabla en pdf-main
CREATE TABLE pdfs (
  pdf_id VARCHAR(50) PRIMARY KEY,
  filename VARCHAR(255),
  status ENUM('pending', 'processing', 'completed', 'failed'),
  markdown_content LONGTEXT,
  extracted_at TIMESTAMP
);

-- Flujo:
1. INSERT INTO pdfs (pdf_id, filename, status) VALUES ('pdf_ABC', 'factura.pdf', 'pending')
2. XADD queue:extraction {pdf_id: 'pdf_ABC', ...}
3. XREAD queue:extraction-results (espera)
4. UPDATE pdfs SET status='completed', markdown_content='...' WHERE pdf_id='pdf_ABC'
```

---

## Configuración en docker-compose.yml

```yaml
services:
  api:
    environment:
      - STREAM_EXTRACTION=queue:extraction
      - STREAM_RESULTS=queue:extraction-results
      - CONSUMER_GROUP=pdf-extractor-group
      - CONSUMER_NAME=pdf-extractor-1
```

---

## Monitoreo

### Ver estado de streams

```bash
# Información del stream de entrada
redis-cli XINFO STREAM queue:extraction

# Resultado:
# - first-entry: {timestamp}
# - last-entry: {timestamp}
# - length: 150 (mensajes pendientes)
# - radix-tree-keys: 1
# - radix-tree-nodes: 2
# - groups: 1 (consumer group)
# - last-generated-id: 1727900500000-0
```

### Ver consumer group

```bash
redis-cli XINFO GROUPS queue:extraction

# Resultado:
# - name: pdf-extractor-group
# - consumers: 2
# - pending: 5 (mensajes sin ACK)
# - last-delivered-id: 1727900450000-0
```

### Ver consumers

```bash
redis-cli XINFO CONSUMERS queue:extraction pdf-extractor-group

# Resultado:
# - pdf-extractor-1
#   - pending: 3
#   - idle: 2000ms
# - pdf-extractor-2
#   - pending: 2
#   - idle: 500ms
```

---

## Manejo de Errores

### Caso: PDF demasiado grande (>20MB)

```
pdf-main envía:
  {pdf_id: "pdf_XYZ", content_base64: "JVBERi0xLjQK..."}

pdf-extractor recibe, rechaza:
  XADD queue:extraction-results {
    pdf_id: "pdf_XYZ",
    status: "failed",
    error: "PDF exceeds max size: 50MB > 20MB"
  }
```

### Caso: Timeout en extracción

```
pdf-extractor (timeout después de 30s):
  XADD queue:extraction-results {
    pdf_id: "pdf_XYZ",
    status: "failed",
    error: "extraction timeout after 30s"
  }
```

### Caso: PDF corrupto

```
pdf-extractor (no puede decodificar):
  XADD queue:extraction-results {
    pdf_id: "pdf_XYZ",
    status: "failed",
    error: "invalid PDF format or corrupted file"
  }
```

---

## Dead Letter Queue (DLQ)

Para mensajes que fallan después de MAX_RETRIES (default: 3):

```bash
redis-cli XADD queue:extraction-dlq "*" \
  pdf_id "pdf_BAD" \
  reason "max_retries_exceeded" \
  last_error "extraction timeout"
```

pdf-main debe monitorear `queue:extraction-dlq` y notificar al usuario.

---

## Validación de Contrato

### Antes de XADD (pdf-main)

```go
func ValidateExtractionMessage(msg map[string]string) error {
	if msg["pdf_id"] == "" {
		return errors.New("pdf_id is required")
	}
	if msg["filename"] == "" {
		return errors.New("filename is required")
	}
	if msg["content_base64"] == "" {
		return errors.New("content_base64 is required")
	}
	if len(msg["content_base64"]) > MAX_BASE64_SIZE {
		return errors.New("content_base64 exceeds max size")
	}
	return nil
}
```

### Antes de XADD (pdf-extractor)

```go
func ValidateResultMessage(msg map[string]string) error {
	if msg["pdf_id"] == "" {
		return errors.New("pdf_id is required")
	}
	if msg["status"] != "success" && msg["status"] != "failed" {
		return errors.New("status must be 'success' or 'failed'")
	}
	if msg["status"] == "success" && msg["markdown_content"] == "" {
		return errors.New("markdown_content required when status is success")
	}
	return nil
}
```