#!/bin/bash

# ============================================================================
# SCRIPT DE VALIDACIÓN - Configuración TP Benchmark
# ============================================================================
# 
# Este script verifica que el entorno esté configurado correctamente para
# ejecutar el benchmark del Trabajo Práctico.
#
# Uso:
#   chmod +x validate-tp-setup.sh
#   ./validate-tp-setup.sh
#

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

ERRORS=0
WARNINGS=0
SUCCESS=0

# ============================================================================
# FUNCIONES DE UTILIDAD
# ============================================================================

print_header() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

print_check() {
    echo -e "${GREEN}✓${NC} $1"
    ((SUCCESS++))
}

print_error() {
    echo -e "${RED}✗${NC} $1"
    ((ERRORS++))
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
    ((WARNINGS++))
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

# ============================================================================
# VERIFICACIONES
# ============================================================================

print_header "VALIDACIÓN DE CONFIGURACIÓN TP - PDF EXTRACTOR"

# --- Docker ---
print_header "1. Verificando Docker"

if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version)
    print_check "Docker instalado: $DOCKER_VERSION"
else
    print_error "Docker no está instalado"
    exit 1
fi

# --- Docker Compose ---
print_header "2. Verificando Docker Compose"

if command -v docker &> /dev/null && docker compose version &> /dev/null; then
    COMPOSE_VERSION=$(docker compose version --short)
    print_check "Docker Compose V2 instalado: v$COMPOSE_VERSION"
else
    print_error "Docker Compose no está instalado o no es V2"
    echo "  Instalar: https://docs.docker.com/compose/install/"
    exit 1
fi

# --- Archivos Requeridos ---
print_header "3. Verificando Archivos Requeridos"

FILES=(
    "docker-compose.tp.yml"
    "docker-compose.tp.with-lb.yml"
    "nginx.conf"
    "TP-BENCHMARK-SETUP.md"
    "pdf-extractor-main/docker-compose.yml"
    "pdf-extractor-main/Dockerfile"
    "pdf-extractor-main/docs/benchmark.js"
)

for file in "${FILES[@]}"; do
    if [ -f "$file" ]; then
        print_check "Existe: $file"
    else
        print_error "Falta: $file"
    fi
done

# --- Validar YAML Syntax ---
print_header "4. Validando Sintaxis de Archivos"

echo -n "  Validando docker-compose.tp.yml..."
if docker compose -f docker-compose.tp.yml config > /dev/null 2>&1; then
    echo -e " ${GREEN}✓${NC}"
    print_check "docker-compose.tp.yml válido"
else
    print_error "docker-compose.tp.yml contiene errores"
fi

echo -n "  Validando docker-compose.tp.with-lb.yml..."
if docker compose -f docker-compose.tp.with-lb.yml config > /dev/null 2>&1; then
    echo -e " ${GREEN}✓${NC}"
    print_check "docker-compose.tp.with-lb.yml válido"
else
    print_error "docker-compose.tp.with-lb.yml contiene errores"
fi

# --- Recursos del Sistema ---
print_header "5. Verificando Recursos del Sistema"

# CPU cores
if [ -f /proc/cpuinfo ]; then
    CPU_COUNT=$(grep -c "^processor" /proc/cpuinfo)
elif command -v sysctl &> /dev/null; then
    CPU_COUNT=$(sysctl -n hw.ncpu)
else
    CPU_COUNT="desconocido"
fi

if [ "$CPU_COUNT" != "desconocido" ]; then
    if [ $CPU_COUNT -ge 4 ]; then
        print_check "CPU cores: $CPU_COUNT (mínimo requerido: 4)"
    else
        print_warning "CPU cores: $CPU_COUNT (mínimo recomendado: 4, evaluación puede ser lenta)"
    fi
else
    print_info "No se pudo detectar CPU cores"
fi

# Memoria
if [ -f /proc/meminfo ]; then
    MEM_KB=$(grep "MemTotal" /proc/meminfo | awk '{print $2}')
    MEM_GB=$((MEM_KB / 1024 / 1024))
elif command -v sysctl &> /dev/null; then
    MEM_BYTES=$(sysctl -n hw.memsize)
    MEM_GB=$((MEM_BYTES / 1024 / 1024 / 1024))
else
    MEM_GB="desconocido"
fi

if [ "$MEM_GB" != "desconocido" ]; then
    if [ $MEM_GB -ge 4 ]; then
        print_check "Memoria RAM: ${MEM_GB}GB (mínimo requerido: 4GB)"
    else
        print_warning "Memoria RAM: ${MEM_GB}GB (mínimo recomendado: 4GB, puede causar OOM)"
    fi
else
    print_info "No se pudo detectar memoria RAM"
fi

# Espacio en disco
DISK_SPACE=$(df . | tail -1 | awk '{print $4}')
DISK_GB=$((DISK_SPACE / 1024 / 1024))

if [ $DISK_GB -ge 5 ]; then
    print_check "Espacio en disco: ${DISK_GB}GB disponibles"
else
    print_warning "Espacio en disco: ${DISK_GB}GB (mínimo recomendado: 5GB)"
fi

# --- Puertos Disponibles ---
print_header "6. Verificando Disponibilidad de Puertos"

PORTS=(
    "80:nginx"
    "8080:api"
    "8081:api-1"
    "8082:api-2"
    "8083:api-3"
    "8084:api-4"
    "8085:api-5"
    "6379:redis"
    "27017:mongodb"
)

for port_info in "${PORTS[@]}"; do
    PORT=$(echo $port_info | cut -d: -f1)
    SERVICE=$(echo $port_info | cut -d: -f2)
    
    if ! netstat -tuln 2>/dev/null | grep -q ":$PORT " && \
       ! lsof -i :$PORT 2>/dev/null | grep -q LISTEN; then
        print_check "Puerto $PORT disponible ($SERVICE)"
    else
        print_warning "Puerto $PORT en uso ($SERVICE) - puede causar conflicto"
    fi
done

# --- Variables de Entorno ---
print_header "7. Verificando Variables de Entorno Clave"

# Buscar en docker-compose.tp.yml
if grep -q "GOMAXPROCS=1" docker-compose.tp.yml; then
    print_check "GOMAXPROCS=1 configurado en TP compose"
else
    print_warning "GOMAXPROCS podría no estar optimizado"
fi

if grep -q "LOG_LEVEL=warn" docker-compose.tp.yml; then
    print_check "LOG_LEVEL=warn configurado para reducir overhead"
else
    print_warning "LOG_LEVEL podría causar overhead en logging"
fi

if grep -q "cpus: '1.0'" docker-compose.tp.yml; then
    print_check "Límites de CPU configurados a 1.0 por contenedor"
else
    print_warning "Límites de CPU podrían no estar configurados"
fi

if grep -q "memory: 512M" docker-compose.tp.yml; then
    print_check "Límites de memoria configurados a 512M"
else
    print_warning "Límites de memoria podrían no estar configurados"
fi

# --- Configuración de Benchmark ---
print_header "8. Verificando Configuración de Benchmark"

if [ -f "pdf-extractor-main/docs/benchmark.js" ]; then
    print_check "Script de benchmark existe"
    
    if grep -q "K6_VUS" docker-compose.tp.yml; then
        VUS=$(grep "K6_VUS" docker-compose.tp.yml | head -1 | sed 's/.*K6_VUS=\([0-9]*\).*/\1/')
        print_check "K6_VUS configurado: $VUS usuarios virtuales"
    fi
    
    if grep -q "K6_DURATION" docker-compose.tp.yml; then
        DURATION=$(grep "K6_DURATION" docker-compose.tp.yml | head -1 | sed 's/.*K6_DURATION=\([^"]*\).*/\1/')
        print_check "K6_DURATION configurado: $DURATION"
    fi
else
    print_error "Script de benchmark no encontrado"
fi

# --- Load Balancer ---
print_header "9. Verificando Configuración de Load Balancer"

if [ -f "nginx.conf" ]; then
    print_check "Configuración nginx existe"
    
    if grep -q "upstream api_backend" nginx.conf; then
        REPLICAS=$(grep -c "server pdf-api-" nginx.conf)
        print_check "Load balancer configurado para $REPLICAS réplicas"
    fi
else
    print_warning "nginx.conf no encontrado - modo sin LB solo"
fi

# --- Docker Daemon ---
print_header "10. Verificando Estado de Docker Daemon"

if docker ps &> /dev/null; then
    print_check "Docker daemon está ejecutándose"
    
    RUNNING=$(docker ps -q | wc -l)
    TOTAL=$(docker ps -a -q | wc -l)
    print_info "Contenedores ejecutándose: $RUNNING/$TOTAL"
else
    print_error "Docker daemon no está disponible"
    echo "  Intenta: sudo systemctl start docker"
    exit 1
fi

# --- Resumen ---
print_header "RESUMEN DE VALIDACIÓN"

echo ""
echo -e "  ${GREEN}✓ Exitosas:${NC}   $SUCCESS"
echo -e "  ${YELLOW}⚠ Advertencias:${NC} $WARNINGS"
echo -e "  ${RED}✗ Errores:${NC}     $ERRORS"
echo ""

if [ $ERRORS -eq 0 ]; then
    echo -e "${GREEN}✓ CONFIGURACIÓN VÁLIDA - Listo para ejecutar benchmark${NC}"
    echo ""
    echo "Próximos pasos:"
    echo "  1. Opción A (sin Load Balancer):"
    echo "     docker compose -f docker-compose.tp.yml up --build"
    echo ""
    echo "  2. Opción B (con Load Balancer):"
    echo "     docker compose -f docker-compose.tp.with-lb.yml up --build"
    echo ""
    exit 0
else
    echo -e "${RED}✗ CONFIGURACIÓN INVÁLIDA - Corregir errores antes de continuar${NC}"
    exit 1
fi