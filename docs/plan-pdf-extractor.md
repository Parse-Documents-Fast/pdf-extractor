# Plan de Desarrollo — PDF Extractor Microservice

## Contexto
Microservicio que absorbe responsabilidades de `pdf-transformator` (ADR-0005). Funciona como **consumer de Redis Streams** (no expone HTTP). Recibe PDFs validados de `pdf-main` vía `queue:extraction` y publica resultados en `queue:extraction-results`.

## Decisión Clave
Nunca aplanar el PDF a texto plano antes de tiempo. Mantener estructura (fuente, posición, tabla) para mapear confiablemente a Markdown en vez de heurísticas sobre texto ya aplanado.

---

## Fase 1: Estructura Base + Redis Consumer

- [ ] Crear estructura: `dev/`, `tests/`, `docs/`
- [ ] Setup: `pyproject.toml`, `.env.example`, `README.md`
- [ ] Configurar FastAPI (solo para health checks/metrics)
- [ ] Configurar `redis-py` para Streams
- [ ] Crear `dev/config.py` con variables de entorno (Redis URI, MongoDB URI, etc)
- [ ] Implementar consumer loop que escucha `queue:extraction`

---

## Fase 2: Extracción Estructurada de PDF

**Elegir librería:** `unipdf` o `pdfcpu` — la que dé mejor acceso a metadata (tamaño de fuente, posición, tablas)

- [ ] Comparar con PDFs de prueba (simple + con tabla)
- [ ] Implementar `ExtractStructure(pdf_bytes)`:
  - Retorna lista de **bloques**, cada uno con:
    - Texto
    - Tamaño de fuente
    - Posición (para detectar tablas)
    - Indicador de celda de tabla
- [ ] Manejo de errores: `PdfExtractionError` (excepción de dominio)
- [ ] Caso bordes: PDF corrupto, sin texto extraíble

---

## Fase 3: Mapeo a Markdown

- [ ] Implementar `MapStructureToMarkdown(blocks)`:
  - Fuente tamaño > umbral → `#` / `##` / `###`
  - Bloques de tabla → sintaxis Markdown (`|...|`)
  - Resto → párrafo plano
- [ ] Ajustar umbrales de tamaño de fuente (iteración con PDFs reales)
- [ ] Tests contra PDFs simple + tabla

---

## Fase 4: Persistencia en MongoDB

- [ ] Modelo `PdfDocument` en MongoDB (con Markdown generado)
- [ ] `PdfRepository` (interfaz) + `MongoPdfRepository` (implementación)
- [ ] Guardar resultado procesado en MongoDB con:
  - `pdf_id`
  - `filename`
  - `markdown_content`
  - `extracted_at`
  - `status` (success/failed)

---

## Fase 5: Redis Streams Integration

- [ ] Consumer que escucha `queue:extraction`:
  - Lee: `pdf_id`, `filename`, `content_base64`
  - Decodifica base64 → bytes
  - Procesa (Extract + Map)
  - Publica en `queue:extraction-results` con Markdown resultado
  - `XACK` al terminar
- [ ] Consumer group setup: `pdf-extractor-group`
- [ ] Manejo de reintentos y dead-letter queue en caso de error

---

## Fase 6: Endpoints Secundarios (Health Check + Metrics)

- [ ] GET `/health` → status del servicio
- [ ] GET `/metrics` → metrics de Prometheus (PDFs procesados, errores, latencia)

---

## Fase 7: Tests + Documentación

- [ ] Tests unitarios: `ExtractStructure`, `MapStructureToMarkdown`
- [ ] Tests de integración: consumer loop + Redis
- [ ] Fixtures: PDFs simple + tabla
- [ ] README con setup, envs, cómo correr
- [ ] Architecture docs en `/docs`

---

## Ya No Aplica
- `pdf-transformator` como repo separado — vive acá como responsabilidad núcleo
- Endpoints REST para PDF upload (eso lo maneja `pdf-main`)
- HTTP como transporte principal (Redis Streams es el contrato)