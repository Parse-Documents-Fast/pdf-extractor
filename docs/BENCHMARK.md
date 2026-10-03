# Benchmark Guide

## Configuración de Benchmark Temporal

El archivo `docker-compose.yml` incluye un servicio `benchmark` que ejecuta pruebas de carga usando k6.

### Correr Benchmark

#### Opción 1: Con docker-compose (todos los servicios)

```bash
docker-compose up -d
docker-compose run --rm benchmark
```

#### Opción 2: Solo benchmark (API ya corriendo)

```bash
docker-compose up benchmark
```

#### Opción 3: Benchmark con configuración personalizada

```bash
docker-compose run -e K6_VUS=20 -e K6_DURATION=60s benchmark
```

### Variables de Entorno

| Variable | Default | Descripción |
|----------|---------|-------------|
| `K6_VUS` | 10 | Usuarios virtuales concurrentes |
| `K6_DURATION` | 30s | Duración del test |
| `K6_ITERATIONS` | - | Número fijo de iteraciones (si se especifica, ignora DURATION) |

### Remover Benchmark

#### Opción 1: Solo el contenedor de benchmark

```bash
docker-compose down benchmark
```

#### Opción 2: Todos los servicios

```bash
docker-compose down
```

#### Opción 3: Incluir volúmenes

```bash
docker-compose down -v
```

### Archivo de Script: `docs/benchmark.js`

Crear `docs/benchmark.js`:

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: __ENV.K6_VUS || 10,
  duration: __ENV.K6_DURATION || '30s',
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'],
    http_req_failed: ['rate<0.1'],
  },
};

export default function () {
  const baseUrl = 'http://api:8080';

  // Test /health endpoint
  const healthRes = http.get(`${baseUrl}/health`);
  check(healthRes, {
    'health status 200': (r) => r.status === 200,
    'health has status field': (r) => r.json('status') === 'healthy',
  });

  // Test /metrics endpoint
  const metricsRes = http.get(`${baseUrl}/metrics`);
  check(metricsRes, {
    'metrics status 200': (r) => r.status === 200,
    'metrics has uptime': (r) => r.json('uptime') !== null,
    'metrics has memory_mb': (r) => r.json('memory_mb') > 0,
    'metrics has requests_total': (r) => r.json('requests_total') >= 0,
  });

  sleep(1);
}
```

### Ejemplos de Uso

#### Benchmark rápido (10 VUs, 30 segundos)

```bash
docker-compose up benchmark
```

#### Benchmark intenso (100 VUs, 5 minutos)

```bash
docker-compose run -e K6_VUS=100 -e K6_DURATION=5m benchmark
```

#### Benchmark con iteraciones fijas (1000 requests)

```bash
docker-compose run -e K6_ITERATIONS=1000 benchmark
```

#### Benchmark sin parar (hasta Ctrl+C)

```bash
docker-compose run -e K6_DURATION=0 benchmark
```

### Interpretar Resultados

El benchmark genera output como:

```
     checks.........................: 100% ✓ 1200      ✗ 0
     data_received..................: 145 kB
     data_sent.......................: 52 kB
     http_req_blocked...............: avg=1.02ms
     http_req_connecting............: avg=0.31ms
     http_req_duration..............: avg=45.23ms p(95)=120ms p(99)=250ms
     http_req_failed................: 0%
     http_req_receiving.............: avg=1.23ms
     http_req_sending..............: avg=0.45ms
     http_req_tls_handshaking.......: avg=0.00ms
     http_req_waiting..............: avg=43.55ms
     http_reqs.......................: 1200 (40/s)
     iteration_duration.............: avg=1.04s
     iterations......................: 1200
```

**Métricas importantes:**
- `http_req_duration` — Latencia promedio
- `http_req_failed` — % de requests fallidos
- `http_reqs` — Requests por segundo
- `checks` — % de checks exitosos

### Troubleshooting

#### "Connection refused"
```bash
# Verificar que API está corriendo
docker-compose logs api

# Esperar a que API inicie
sleep 10
docker-compose up benchmark
```

#### "Module not found"
```bash
# Asegurar que docs/benchmark.js existe
ls -la docs/benchmark.js

# O crear directorio y archivo
mkdir -p docs
# Crear benchmark.js con contenido anterior
```

#### Benchmark no termina
```bash
# Abrir otro terminal
docker-compose down benchmark

# O matar el contenedor
docker-compose kill benchmark
```

### Automatizar Benchmark en CI/CD

#### GitHub Actions

```yaml
- name: Run Benchmark
  run: |
    docker-compose up -d
    sleep 10
    docker-compose run -e K6_VUS=5 -e K6_DURATION=10s benchmark
    docker-compose down
```

#### GitLab CI

```yaml
benchmark:
  stage: test
  services:
    - docker:dind
  script:
    - docker-compose up -d
    - sleep 10
    - docker-compose run -e K6_VUS=5 -e K6_DURATION=10s benchmark
    - docker-compose down
```

### Limpiar Recursos

```bash
# Remover contenedor parado
docker-compose rm benchmark

# Remover imagen (si existe)
docker rmi grafana/k6:latest

# Limpiar volúmenes no usados
docker volume prune -f

# Limpiar todo (contenedores, redes, volúmenes)
docker-compose down -v
```