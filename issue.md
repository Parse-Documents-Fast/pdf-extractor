# Problema

Este microservicio falla por varias razones, pero el problema de fondo **es que se esta intentando remplazar las funciones de otros microservicios** se esta intentando construir un sistema paralelo. 

## Desglose

### Generales

1. Un `Nginx.con` adentro de `internal/config`/ de un proyecto Go, cuando toda la arquitectura del equipo está parada sobre Traefik. Y no es solo un archivo huérfano,`docker-compose.tp.with-lb.yml `arma un balanceador nginx completo con 5 réplicas manuales (api-1 a api-5, puertos 8081-8085), **reinventando exactamente lo que Traefik + el deploy.replicas que ya definieron en ADR-0006 hacen solos.** Es doble trabajo para lograr lo mismo que el resto del sistema ya resuelve. Nginx y Traefik resuelven una misma cosa, no se usan juntos, y funcionan a nivel macro, no a nivel microservicios.

2. `cmd/api/main.go` y `cmd/pdf-extractor/main.go`, dos binarios distintos en el mismo repo. Si api es el intento de exponer el handler síncrono del TP, no tendría que ser un segundo main con su propio servidor HTTP completo. **En condiciones normales un repo de Go nunca genera mas de un artefacto, y mas en microservicios.**

3. Hay 3 carpetas de test: `test`, `testdata`, `tests`, las 3 están mal y no cumplen con lo esperado. En Go los test se manejan a nivel paquete, `extractor.go` y a lado su `extractor_test.go`. La carpeta `test/` se reveserva para stubs y tests mas complejos.

### Arquitectura y alcance

Aca esta lo que hace el microservicio inviable:

1. Su propia Mongo (`MONGO_DB=pdf-extractor`, `MONGO_COLLECTION=documents`), totalmente separada de la Mongo real que usa pdf-persistance. Si esto se despliega así, los resultados de extracción quedan en una base de datos que nadie más del sistema puede leer. No hay forma de que pdf-main vea ese resultado. **Responsabilidad de pdf-peristance**

2. Su propio Redis (allkeys-lru, puerto 6379 expuesto al host), **no es la instancia queue de pdf-infra** (que se defininio como noeviction + AOF, justamente porque perder un job de extracción es inaceptable). Con allkeys-lru, bajo presión de memoria, Redis puede borrar jobs en cola para hacer lugar, exactamente el escenario que ADR-0004/0005 estaban diseñados para evitar. **Responsabilidad de pdf-main**

3. Un servicio api con `healthcheck` HTTP, puertos expuestos, contradice ADR-0004: pdf-extractor no debería tener superficie HTTP en absoluto (salvo el handler síncrono puntual para el TP, que además debería vivir como parte del hexágono existente, no como un segundo servicio "api" separado dentro del mismo repo)

4. Red propia (`pdf-network`, `172.20.0.0/16`), ni siquiera está en `fast_pdf_network`. Tal como está, **este contenedor no puede hablarle a pdf-main ni a nada del resto del sistema, aunque quisiera.**

## Conclusión

No he visto la logica interna escrita en Go, pero con lo resaltado ya se tiene que volver a comenzar: crear un `spec.md` que entienda su responsabilidad y alcances. Este repo quedo grande no por que lo sea, la logica de extraer  un pdf no es grande, el problema esta en que intenta resolver aspectos de MongoDB, Redis y balanceador de carga que ni siquiera le corresponden. El tema de replicas se maneja con un override en un .yml, no introduciendo una tecnología nueva. 
 