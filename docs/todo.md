# TODO — PDF Extractor

## Setup Inicial
- [ ] Inicializar repositorio con estructura base
- [ ] Crear `pyproject.toml` con dependencias (redis, beanie, motor, pdfcpu/unipdf)
- [ ] Configurar `.env` y `.env.example` (REDIS_URI, MONGO_URI, etc)
- [ ] Setup FastAPI (solo para health checks)

## Librería PDF
- [ ] Comparar `unipdf` vs `pdfcpu` con PDFs reales
- [ ] Elegir la que mejor acceso da a metadata (fuente, posición, tablas)

## Extracción de Estructura
- [ ] Modelo `Block` (texto, fuente, posición, tabla_celda)
- [ ] Función `extract_structure(pdf_bytes) -> List[Block]`
- [ ] Manejo de errores: `PdfExtractionError`
- [ ] Tests con PDF simple + tabla

## Mapeo a Markdown
- [ ] Función `map_structure_to_markdown(blocks) -> str`
- [ ] Reglas: tamaño_fuente > umbral → headers, tablas → sintaxis MD, resto → párrafo
- [ ] Ajustar umbrales iterativamente
- [ ] Tests contra PDFs reales

## Persistencia MongoDB
- [ ] Modelo `PdfDocument` con campos: pdf_id, filename, markdown_content, extracted_at, status
- [ ] `PdfRepository` (interfaz)
- [ ] `MongoPdfRepository` (implementación)

## Redis Streams Consumer
- [ ] Consumer que escucha `queue:extraction`
- [ ] Parser: pdf_id, filename, content_base64
- [ ] Decodificar base64 → bytes
- [ ] Procesar (Extract + Map)
- [ ] Publicar resultado en `queue:extraction-results`
- [ ] `XACK` al terminar
- [ ] Setup consumer group: `pdf-extractor-group`
- [ ] Manejo de reintentos y errores

## Health Check + Metrics
- [ ] GET `/health`
- [ ] GET `/metrics` (Prometheus)

## Tests
- [ ] Tests unitarios: `extract_structure`, `map_structure_to_markdown`
- [ ] Tests de integración: consumer + Redis
- [ ] Fixtures: PDFs simple + tabla

## Documentación
- [ ] README.md con setup e instrucciones
- [ ] .env.example con todas las variables
- [ ] Architecture docs en `/docs`
- [ ] Contrato Redis Streams documentado