#!/bin/bash

# ============================================================================
# SCRIPT DE BENCHMARK - Evaluación TP PDF Extractor
# ============================================================================
#
# Ejecuta el benchmark del Trabajo Práctico de forma automatizada.
# 
# Uso:
#   chmod +x run-tp-benchmark.sh
#   ./run-tp-benchmark.sh                    # Modo default (spike, sin LB)
#   ./run-tp-benchmark.sh --with-lb          # Con load balancer
#   ./run-tp-benchmark.sh --load              # Load test (más agresivo)
#   ./run-tp-benchmark.sh --throughput        # Throughput test (máximo)
#

set -e

# ============================================================================
# CONFIGURACIÓN
# ============================================================================

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Modo por defecto
MODE="spike"
WITH_LB=false
COMPOSE_FILE="docker-compose.tp.yml"
KEEP_RUNNING=false

# Parámetros de benchmark por modo
declare -A MODES
MODES[spike]="10 30s"           # 10 VUs, 30 segundos (default spike test)
MODES[load]="50 60s"            # 50 VUs, 60 segundos (load test)
MODES[throughput]="100 120s"    # 100 VUs, 2 minutos (max throughput)

# ============================================================================
# FUNCIONES
# ============================================================================

print_header() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

print_info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

show_usage() {
    cat << EOF
${BLUE}PDF EXTRACTOR - Script de Benchmark TP${NC}

${YELLOW}Uso:${NC}
  ./run-tp-benchmark.sh [OPTIONS]

${YELLOW}Opciones:${NC}
  --spike           Spike test (10 VUs, 30s) - DEFECTO
  --load            Load test (50 VUs, 60s)
  --throughput      Throughput test (100 VUs, 120s)
  --with-lb         Usar load balancer (nginx)
  --keep-running    No eliminar contenedores después
  --help            Mostrar esta ayuda

${YELLOW}Ejemplos:${NC}
  # Spike test sin load balancer
  ./run-tp-benchmark.sh --spike

  # Load test con load balancer
  ./run-tp-benchmark.sh --load --with-lb

  # Throughput test, mantener servicios ejecutándose
  ./run-tp-benchmark.sh --throughput --keep-running

${YELLOW}Modos de Benchmark:${NC}
  spike      : Prueba rápida de carga (default)
  load       : Prueba de carga moderada
  throughput : Prueba de máximo throughput

EOF
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --spike)
                MODE="spike"
                shift
                ;;
            --load)
                MODE="load"
                shift
                ;;
            --throughput)
                MODE="throughput"
                shift
                ;;
            --with-lb)
                WITH_LB=true
                shift
                ;;
            --keep-running)
                KEEP_RUNNING=true
                shift
                ;;
            --help|-h)
                show_usage
                exit 0
                ;;
            *)
                print_error "Opción desconocida: $1"
                show_usage
                exit 1
                ;;
        esac
    done

    # Seleccionar archivo compose
    if [ "$WITH_LB" = true ]; then
        COMPOSE_FILE="docker-compose.tp.with-lb.yml"
    fi
}

verify_setup() {
    print_header "Verificando Setup"
    
    # Verificar archivo compose
    if [ ! -f "$COMPOSE_FILE" ]; then
        print_error "Archivo no encontrado: $COMPOSE_FILE"
        exit 1
    fi
    print_success "Archivo compose encontrado: $COMPOSE_FILE"
    
    # Validar sintaxis
    print_info "Validando sintaxis de compose..."
    if ! docker compose -f "$COMPOSE_FILE" config > /dev/null 2>&1; then
        print_error "Archivo compose inválido"
        exit 1
    fi
    print_success "Sintaxis válida"
    
    # Verificar script de benchmark
    if [ ! -f "pdf-extractor-main/docs/benchmark.js" ]; then
        print_error "Script de benchmark no encontrado"
        exit 1
    fi
    print_success "Script de benchmark disponible"
    
    # Verificar espacio en disco
    print_info "Verificando espacio en disco..."
    DISK_AVAILABLE=$(df . | tail -1 | awk '{print $4}')
    DISK_GB=$((DISK_AVAILABLE / 1024 / 1024))
    
    if [ $DISK_GB -lt 2 ]; then
        print_warning "Espacio limitado: ${DISK_GB}GB"
    else
        print_success "Espacio disponible: ${DISK_GB}GB"
    fi
}

cleanup() {
    print_info "Limpiando recursos..."
    
    # Eliminar contenedores
    docker compose -f "$COMPOSE_FILE" down -v 2>/dev/null || true
    
    print_success "Limpieza completada"
}

startup() {
    print_header "Iniciando Infraestructura"
    
    print_info "Levantando servicios (esto puede tomar 30-60 segundos)..."
    
    # Usar --build para asegurar imágenes actualizadas
    docker compose -f "$COMPOSE_FILE" up -d --build
    
    print_success "Servicios iniciados"
    
    # Esperar a que API esté saludable
    print_info "Esperando a que API esté saludable..."
    WAIT_TIME=0
    MAX_WAIT=120
    
    while [ $WAIT_TIME -lt $MAX_WAIT ]; do
        if [ "$WITH_LB" = true ]; then
            # Con LB, verificar puerto 80
            if curl -s http://localhost:80/health > /dev/null 2>&1; then
                print_success "API saludable (vía LB)"
                return 0
            fi
        else
            # Sin LB, verificar puerto 8080
            if curl -s http://localhost:8080/health > /dev/null 2>&1; then
                print_success "API saludable"
                return 0
            fi
        fi
        
        sleep 2
        WAIT_TIME=$((WAIT_TIME + 2))
        echo -ne "\r  Esperando... ${WAIT_TIME}s / ${MAX_WAIT}s"
    done
    
    echo ""
    print_error "Timeout esperando API saludable"
    print_info "Logs:"
    docker compose -f "$COMPOSE_FILE" logs api | tail -20
    exit 1
}

run_benchmark() {
    print_header "Ejecutando Benchmark: $MODE"
    
    # Extraer parámetros del modo
    read -r VUS DURATION <<< "${MODES[$MODE]}"
    
    echo -e "  ${CYAN}Modo:${NC}        $MODE"
    echo -e "  ${CYAN}VUs:${NC}         $VUS"
    echo -e "  ${CYAN}Duración:${NC}    $DURATION"
    if [ "$WITH_LB" = true ]; then
        echo -e "  ${CYAN}Endpoint:${NC}   http://localhost/health (vía nginx LB)"
        echo -e "  ${CYAN}Réplicas:${NC}   5"
    else
        echo -e "  ${CYAN}Endpoint:${NC}   http://localhost:8080/health"
        echo -e "  ${CYAN}Réplicas:${NC}   1"
    fi
    echo ""
    
    # Ejecutar benchmark
    docker compose -f "$COMPOSE_FILE" run --rm \
        -e K6_VUS="$VUS" \
        -e K6_DURATION="$DURATION" \
        benchmark
}

collect_metrics() {
    print_header "Recolectando Métricas"
    
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    METRICS_DIR="benchmark-results-$TIMESTAMP"
    mkdir -p "$METRICS_DIR"
    
    # Logs del API
    print_info "Recolectando logs del API..."
    docker compose -f "$COMPOSE_FILE" logs api > "$METRICS_DIR/api-logs.txt" 2>&1 || true
    
    # Stats de Docker
    print_info "Recolectando estadísticas de Docker..."
    docker stats --no-stream > "$METRICS_DIR/docker-stats.txt" 2>&1 || true
    
    # Métricas del API
    if [ "$WITH_LB" = true ]; then
        curl -s http://localhost/metrics > "$METRICS_DIR/api-metrics.json" 2>&1 || true
    else
        curl -s http://localhost:8080/metrics > "$METRICS_DIR/api-metrics.json" 2>&1 || true
    fi
    
    # Información del sistema
    echo "Docker version:" > "$METRICS_DIR/system-info.txt"
    docker --version >> "$METRICS_DIR/system-info.txt"
    echo "" >> "$METRICS_DIR/system-info.txt"
    echo "Docker Compose version:" >> "$METRICS_DIR/system-info.txt"
    docker compose version >> "$METRICS_DIR/system-info.txt" || true
    
    print_success "Métricas guardadas en: $METRICS_DIR"
    
    # Mostrar resumen
    echo ""
    echo "📊 ${CYAN}Archivos de Resultado:${NC}"
    ls -lh "$METRICS_DIR"
}

show_results_summary() {
    print_header "Resumen de Benchmark"
    
    if [ -f "$METRICS_DIR/docker-stats.txt" ]; then
        echo -e "${CYAN}📈 Estadísticas de Recursos:${NC}"
        cat "$METRICS_DIR/docker-stats.txt" | head -20
        echo ""
    fi
    
    if [ -f "$METRICS_DIR/api-metrics.json" ]; then
        echo -e "${CYAN}📊 Métricas de API:${NC}"
        cat "$METRICS_DIR/api-metrics.json" | jq . 2>/dev/null || \
        cat "$METRICS_DIR/api-metrics.json"
        echo ""
    fi
}

# ============================================================================
# MAIN
# ============================================================================

main() {
    parse_args "$@"
    
    print_header "PDF EXTRACTOR - Benchmark TP"
    echo -e "Modo: ${GREEN}${MODE}${NC}"
    if [ "$WITH_LB" = true ]; then
        echo -e "Load Balancer: ${GREEN}Habilitado${NC}"
    else
        echo -e "Load Balancer: ${YELLOW}Deshabilitado${NC}"
    fi
    echo ""
    
    # Validar
    verify_setup
    
    # Trap para limpieza en caso de error
    trap cleanup EXIT
    
    # Ejecutar flujo
    startup
    run_benchmark
    collect_metrics
    show_results_summary
    
    # Decidir si mantener servicios corriendo
    if [ "$KEEP_RUNNING" = true ]; then
        echo ""
        print_warning "Servicios mantienen ejecución (--keep-running)"
        print_info "Para detener: docker compose -f $COMPOSE_FILE down"
        # No ejecutar cleanup
        trap - EXIT
    else
        echo ""
        print_info "Deteniendo servicios..."
    fi
    
    print_header "✓ Benchmark Completado"
    echo "Resultados: $METRICS_DIR"
}

# ============================================================================
# PUNTO DE ENTRADA
# ============================================================================

main "$@"