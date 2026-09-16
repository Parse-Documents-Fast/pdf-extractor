# plan.md — pdf-extractor

## Qué hay que construir
Recibe los bytes de un PDF ya validado y devuelve su contenido **estructurado** — no texto plano. La decisión clave de este servicio: nunca aplanar el PDF a un string antes de tiempo, porque esa información (tamaño de fuente, posición, bordes de tabla) es exactamente lo que `pdf-transformator` necesita después para reconstruir Markdown de forma confiable.

## Cómo construirlo, en orden
1. Elegir entre `unipdf` y `pdfcpu` — la que dé mejor acceso a metadata de fuente/posición/tabla, probando con un par de PDFs reales de prueba (uno simple, uno con tabla).
2. Definir el shape del JSON de salida: una lista de bloques, cada uno con su texto, tamaño de fuente, y si pertenece a una tabla (y en qué celda).
3. Implementar la extracción sobre ese shape — sin intentar todavía convertir nada a Markdown, eso es trabajo de `pdf-transformator`.
4. Manejar el caso de PDF corrupto o sin texto extraíble con una excepción de dominio propia, no un string vacío silencioso (mismo error que ya existía en el monolito, `PdfExtractionError`, adaptado a Go).

## Depende de
Que el shape del JSON estructurado esté acordado con quien construya `pdf-transformator` antes de arrancar — es el contrato más importante de todo el pipeline de PDF.

## Actualización — transporte (ADR-0004)
Este servicio ya no expone un endpoint HTTP — es un consumer de Redis Streams. Consume jobs de `queue:extraction` (grupo de consumidores, con `XACK` al terminar cada job), y publica el resultado en `queue:extraction-results` (éxito o error, según ADR-0001). La función núcleo de extracción (`ExtractStructure`, por ADR-0004) no cambia — solo cambia el adaptador que la llama, que pasa de ser un handler HTTP a ser un loop de consumo de stream.
