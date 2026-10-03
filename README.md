# 📄 PDF Extractor

Extrae contenido de PDFs a Markdown estructurado. Procesa automáticamente documentos complejos, detecta títulos y tablas, y almacena en MongoDB.

**Rendimiento:** 4-5 PDFs/segundo por instancia  
**Almacenamiento:** MongoDB (con índices)  
**Cola:** Redis Streams (con reintentos automáticos)

## ⚡ Setup en 3 minutos

```bash
git clone https://github.com/PARSE-DOCUMENT-FAST/pdf-extractor.git
cd pdf-extractor

# Iniciar (incluye API, Consumer, Redis, MongoDB)
docker-compose up -d

# Verificar que funciona
curl http://localhost:8080/health
```

Listo. Tu API está en `http://localhost:8080`

## 🔌 API Endpoints

```bash
# Health check (Kubernetes probes, monitoreo)
curl http://localhost:8080/health
# {"status":"healthy","message":"PDF Extractor API is running","time":"..."}

# Métricas (Prometheus, dashboards)
curl http://localhost:8080/metrics
# {"uptime":"2h30m","goroutines":12,"memory_mb":45.32,"requests_total":1250}
```

## 📖 Documentación

- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** — Por qué Markdown, diagrama del pipeline, decisiones de diseño
- **[REDIS_CONTRACT.md](docs/REDIS_CONTRACT.md)** — Formato de mensajes, ejemplos Python/Go
- **[BENCHMARK.md](docs/BENCHMARK.md)** — Cómo correr load tests con k6

## 🛠️ Operaciones

```bash
docker-compose ps                    # Ver estado
docker-compose logs -f api           # Logs del API
docker-compose logs -f consumer      # Logs del worker
docker-compose restart consumer      # Reiniciar worker
docker-compose down                  # Parar todo
docker-compose up -d                 # Levantar de nuevo
```

## ⚙️ Configuración

Edita `docker-compose.yml`:

```yaml
# Worker (procesa PDFs)
WORKER_CONCURRENCY: 4                # Más = más rápido (consume CPU/RAM)
EXTRACTION_TIMEOUT: 60s              # Para PDFs complejos
PDF_MAX_BYTES: 52428800              # Aumenta a 50MB si necesitas

# Extracción (detecta títulos)
HEADING_RATIO_H1: 1.8                # Font ratio para H1
HEADING_RATIO_H2: 1.4                # Font ratio para H2
```

## ❓ FAQ

**¿Cuánto tarda procesar un PDF?**  
2-5 segundos típicamente. Ver `/metrics` para latencia real.

**¿Tamaño máximo?**  
20MB por defecto. Aumenta `PDF_MAX_BYTES` en docker-compose.yml

**¿Dónde se guardan los PDFs procesados?**  
En MongoDB. Acceso: `docker-compose exec mongo mongosh` → `use pdf-extractor` → `db.documents.find()`

**¿Qué pasa si un PDF falla?**  
Reintentos automáticos (3 veces). Si sigue fallando, va a Dead Letter Queue. Ver logs del consumer.

**¿Puedo procesar más PDFs en paralelo?**  
Sí, aumenta `WORKER_CONCURRENCY`. Pero sube también CPU/RAM del container.

---

**MIT License** • Procesamiento de documentos a escala  
[GitHub](https://github.com/PARSE-DOCUMENT-FAST/pdf-extractor) • [Issues](https://github.com/PARSE-DOCUMENT-FAST/pdf-extractor/issues)