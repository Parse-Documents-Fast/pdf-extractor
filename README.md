# 📄 PDF Extractor

Microservicio asíncrono en Go diseñado para transformar documentos PDF en Markdown estructurado de forma rápida y eficiente, integrándose perfectamente en arquitecturas distribuidas mediante Redis Streams.

## 🌟 Características

- **Procesamiento Asíncrono:** Consume jobs desde `queue:extraction` y publica resultados en `queue:extraction-results`.
- **Markdown Canónico:** Detección inteligente de encabezados y tablas basada en coordenadas y tamaño de fuente.
- **Cero Persistencia Local:** Sin bases de datos propias; delegando el almacenamiento de forma limpia.
- **Resiliencia:** Reintentos automáticos con backoff exponencial y derivación a Dead Letter Queue (DLQ).
- **Observabilidad:** Endpoints nativos de salud (`/health`) y métricas (`/metrics`).

## 🚀 Puesta en Marcha

### 1. Variables de Entorno
Configura tu entorno basándote en el ejemplo:
```bash
cp .env.example .env
```

### 2. Ejecución con Docker
Levanta el servicio y su infraestructura de soporte en segundos:
```bash
docker compose up --build -d
```

### 3. Desarrollo local (Makefile)
```bash
make build   # Compila el binario
make test    # Ejecuta los tests
make run     # Ejecuta el servicio
```

## 📖 Documentación y Comandos

Para una guía detallada de comandos de operación y pruebas, consulta el archivo [commands.md](commands.md). El spec técnico completo se encuentra en [docs/spec.md](docs/spec.md).

---
Hecho con pasión por el equipo. Licencia MIT.
