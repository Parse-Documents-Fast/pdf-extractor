# 📊 Configuración de Evaluación del Trabajo Práctico

## ⚡ Quick Start

```bash
# 1️⃣ Validar configuración
chmod +x validate-tp-setup.sh
./validate-tp-setup.sh

# 2️⃣ Ejecutar benchmark
chmod +x run-tp-benchmark.sh
./run-tp-benchmark.sh --spike          # Spike test (default)
./run-tp-benchmark.sh --load --with-lb # Load test con LB
```

---

## 📁 Archivos Incluidos

### Archivos de Compose

| Archivo | Descripción | Cuándo Usar |
|---------|-------------|------------|
| `docker-compose.yml` | Original (desarrollo) | Iteración rápida, debugging |
| `docker-compose.tp.yml` | **Evaluación sin LB** | Benchmark simple (1 réplica) |
| `docker-compose.tp.with-lb.yml` | **Evaluación con LB** | Benchmark con 5 réplicas |

### Archivos de Configuración

| Archivo | Descripción |
|---------|-------------|
| `nginx.conf` | Configuración del load balancer (5 réplicas) |
| `validate-tp-setup.sh` | Script de validación del entorno |
| `run-tp-benchmark.sh` | Script de ejecución automatizada |

### Archivos de Documentación

| Archivo | Descripción |
|---------|-------------|
| `TP-BENCHMARK-SETUP.md` | Guía detallada (50+ páginas) |
| `README.TP.md` | Este archivo (resumen ejecutivo) |

---

## 🎯 Configuración de Recursos

### Límites por Contenedor

```yaml
api:
  cpus: '1.0'        # 1 CPU
  memory: 512M       # 512 MB
  
redis:
  cpus: '0.5'        # 0.5 CPU
  memory: 256M       # 256 MB
  
mongo:
  cpus: '1.0'        # 1 CPU
  memory: 512M       # 512 MB
```

### Escalado Horizontal

**Sin Load Balancer:**
- 1 réplica en puerto 8080
- Comando: `docker compose -f docker-compose.tp.yml scale api=3`

**Con Load Balancer:**
- 5 réplicas en puertos 8081-8085
- LB en puerto 80 con distribución `least_conn`
- Comando: `docker compose -f docker-compose.tp.with-lb.yml up --build`

---

## 🔧 Uso

### Opción 1: Sin Load Balancer (Simple)

```bash
# Levantar
docker compose -f docker-compose.tp.yml up --build

# Ejecutar benchmark (en otra terminal)
docker compose -f docker-compose.tp.yml run -e K6_VUS=10 -e K6_DURATION=30s benchmark

# Detener
docker compose -f docker-compose.tp.yml down
```

**Endpoints:**
- API: `http://localhost:8080`
- Health: `http://localhost:8080/health`

---

### Opción 2: Con Load Balancer (Escalable)

```bash
# Levantar (5 réplicas)
docker compose -f docker-compose.tp.with-lb.yml up --build

# Ejecutar benchmark (en otra terminal)
docker compose -f docker-compose.tp.with-lb.yml run benchmark

# Detener
docker compose -f docker-compose.tp.with-lb.yml down
```

**Endpoints:**
- LB: `http://localhost` (puerto 80)
- API-1: `http://localhost:8081`
- API-2: `http://localhost:8082`
- ... API-5: `http://localhost:8085`
- Health: `http://localhost/health`

---

### Opción 3: Script Automatizado

```bash
# Spike test (default)
./run-tp-benchmark.sh

# Load test con LB
./run-tp-benchmark.sh --load --with-lb

# Throughput test
./run-tp-benchmark.sh --throughput

# Mantener servicios ejecutándose
./run-tp-benchmark.sh --keep-running
```

**Modos:**
- `spike`: 10 VUs, 30s (rápido)
- `load`: 50 VUs, 60s (moderado)
- `throughput`: 100 VUs, 120s (intenso)

---

## 📊 Variables de Benchmark

```yaml
# En docker-compose.tp.yml
benchmark:
  environment:
    - K6_VUS=10            # Usuarios virtuales
    - K6_DURATION=30s      # Duración
    - K6_ITERATIONS=1000   # O iteraciones fijas
```

---

## ✅ Checklist Previo

- [ ] Docker & Docker Compose v2 instalados
- [ ] 4+ cores CPU disponibles
- [ ] 4+ GB RAM disponibles
- [ ] 5+ GB espacio en disco
- [ ] Puertos 80, 8080, 6379, 27017 disponibles
- [ ] Ejecutar validación: `./validate-tp-setup.sh`

---

## 🔍 Verificación

### Health Check

```bash
# Sin LB
curl http://localhost:8080/health

# Con LB
curl http://localhost/health
```

**Respuesta esperada:**
```json
{"status":"healthy","uptime":123,"memory_mb":256,"requests_total":1500}
```

### Métricas de Recursos

```bash
# Ver consumo en tiempo real
watch -n 1 'docker stats --no-stream'

# Ver logs
docker compose -f docker-compose.tp.yml logs -f api
```

---

## 🧹 Limpieza

```bash
# Detener servicios
docker compose -f docker-compose.tp.yml down

# Limpiar volúmenes (reset de datos)
docker compose -f docker-compose.tp.yml down -v

# Limpiar todo (contenedores + imágenes)
docker compose -f docker-compose.tp.yml down -v --remove-orphans --rmi all
```

---

## 🐛 Troubleshooting

### "Port already in use"

```bash
# Identificar qué está usando el puerto
lsof -i :8080

# O cambiar puerto en compose
sed -i 's/8080:8080/8081:8080/' docker-compose.tp.yml
```

### "OOMKilled" (out of memory)

```bash
# Aumentar límite en compose (no recomendado)
sed -i 's/512M/1G/g' docker-compose.tp.yml

# O liberar memoria del host
docker system prune -a --volumes
```

### "Connection refused"

```bash
# Esperar más tiempo a que servicios inicien
sleep 30

# Ver logs
docker compose logs redis mongo api
```

---

## 📈 Resultados Esperados

### Spike Test (10 VUs, 30s)

```
checks.........................: 100% ✓ 300    ✗ 0
data_received..................: 36 kB
data_sent.......................: 13 kB
http_req_duration..............: avg=45ms p(95)=120ms p(99)=250ms
http_req_failed................: 0%
http_reqs.......................: 300 (10/s)
iterations......................: 300
```

### Load Test (50 VUs, 60s)

```
checks.........................: 100% ✓ 3000  ✗ 0
http_req_duration..............: avg=150ms p(95)=300ms p(99)=500ms
http_reqs.......................: 3000 (50/s)
```

### Throughput Test (100 VUs, 120s)

```
checks.........................: 100% ✓ 6000  ✗ 0
http_req_duration..............: avg=250ms p(95)=600ms p(99)=1000ms
http_reqs.......................: 6000 (50/s)
```

---

## 📚 Documentación Adicional

Para más detalles, ver:

- **Guía Completa:** `TP-BENCHMARK-SETUP.md`
- **Docker Docs:** https://docs.docker.com/compose/
- **K6 Docs:** https://k6.io/docs/
- **Nginx Docs:** https://nginx.org/en/docs/

---

## 🤝 Soporte

1. Ejecutar validación: `./validate-tp-setup.sh`
2. Revisar logs: `docker compose logs`
3. Contactar docentes con output de validación

---

## 📝 Tareas Implementadas

✅ Crear `docker-compose.tp.yml` con límites de recursos  
✅ Configurar `cpus: '1.0'` y `memory: 512M` por contenedor  
✅ Crear `docker-compose.tp.with-lb.yml` con 5 réplicas  
✅ Configurar nginx load balancer con `least_conn`  
✅ Optimizar variables de entorno (LOG_LEVEL, GOMAXPROCS)  
✅ Crear scripts de validación y ejecución  
✅ Documentación completa  

---

**Versión:** 1.0  
**Fecha:** Octubre 2, 2026  
**Autor:** Equipo de Cátedra