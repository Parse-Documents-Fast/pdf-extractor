# plan.md — pdf-extractor

## Qué hay que construir
Absorbe lo que iba a ser `pdf-transformator` (ADR-0005). Recibe los bytes de un PDF ya validado y devuelve **Markdown** — dos funciones núcleo internas, sin salto de red entre ellas: extraer estructura (texto + tamaño de fuente + posición/tabla) y mapear esa estructura a Markdown. La decisión clave sigue siendo la misma: nunca aplanar el PDF a texto plano antes de tiempo, porque esa información estructural es la que permite mapear a Markdown de forma confiable en vez de heurística sobre texto ya aplanado.

## Cómo construirlo, en orden
1. Elegir entre `unipdf` y `pdfcpu` — la que dé mejor acceso a metadata de fuente/posición/tabla, probando con un par de PDFs reales de prueba (uno simple, uno con tabla).
2. `ExtractStructure()`: extraer una lista de bloques, cada uno con su texto, tamaño de fuente, y si pertenece a una tabla (y en qué celda).
3. `EstructuraAMarkdown()`: mapear esos bloques a sintaxis Markdown — tamaño de fuente por encima de cierto umbral → `#`/`##`; bloques de tabla → sintaxis de tabla Markdown (`|...|`); el resto → párrafo plano.
4. Probar contra los mismos PDFs de prueba (simple + con tabla) y ajustar los umbrales de tamaño de fuente según lo que salga mal — es la parte que más iteración va a necesitar.
5. Manejar el caso de PDF corrupto o sin texto extraíble con una excepción de dominio propia, no un string vacío silencioso (`PdfExtractionError`, adaptado a Go).

## Transporte (ADR-0004 + ADR-0005)
Consumer de Redis Streams, no expone HTTP. Consume jobs de `queue:extraction` (consumer group, `XACK` al terminar), publica el resultado en `queue:extraction-results` — ahora con el Markdown ya armado, no una estructura intermedia ni HTML.

## Ya no aplica
`pdf-transformator` como repo separado — se descarta, esta responsabilidad vive acá adentro como una segunda función núcleo, no como otro servicio de red.
