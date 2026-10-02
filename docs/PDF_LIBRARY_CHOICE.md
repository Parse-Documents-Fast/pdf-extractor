# Elección de librería Go para extraer estructura de PDF

**Estado:** Aceptado

## Contexto

El `pdf-extractor` convierte PDFs a Markdown preservando estructura: tamaño de fuente (para headers), posición X/Y (para tablas por alineación de coordenadas) y página. A diferencia de un extractor de "texto plano", necesitamos acceder a la **fuente y las coordenadas de cada pieza de texto**, no solo al contenido.

Restricciones heredadas de los ADR:

- ADR-0002: todo en RAM, sin tocar disco (el binario llega por `content_base64`).
- ADR-0004: el núcleo de extracción debe ser una función pura (`ExtractStructure`).
- La librería corre dentro del consumer: un PDF malformado no debe tumbar el proceso.

## Criterios de evaluación

1. **Licencia compatible** con un repo de la organización (BSD/MIT/Apache; NO AGPL, NO comercial con API key).
2. **Acceso a fuente y coordenadas** (font name, font size, X, Y), no solo texto plano.
3. **Sin cgo** si es posible (imagen Docker y CI más simples).
4. **Robustez** con PDFs corruptos y **mantenimiento** activo.

## Candidatos

| Candidato | Licencia | Fuente/posición | cgo | Mantenimiento | Veredicto |
|---|---|---|---|---|---|
| unidoc/unipdf | Comercial/AGPL + API key | Sí | No | Activo | Rechazado (licencia) |
| pdfcpu | Apache-2.0 | No (texto plano) | No | Activo | Rechazado (sin posición por glifo) |
| pdflib (PDFlib) | Comercial (C) | Sí | cgo | Activo | Rechazado (comercial + cgo) |
| go-pdfium | Apache-2.0 / PDFium BSD | Sí | cgo/WASM | Activo | Rechazado (build pesado) |
| go-fitz (MuPDF) | MuPDF AGPL | Sí | cgo | Activo | Rechazado (AGPL) |
| **ledongthuc/pdf** (fork de rsc.io/pdf) | BSD-3-Clause | Sí (`Text{Font, FontSize, X, Y, W, S}`) | No | Comunidad (625★) | **Elegido** |
| dslipak/pdf (fork de rsc.io/pdf) | BSD-3-Clause | Sí (mismo tipo) | No | Comunidad (92★) | Alternativa |

## Evaluación (spike)

`research/compare_libs.go` carga los fixtures de `testdata/` con `ledongthuc/pdf` vía `pdf.NewReader(bytes.NewReader(data), len(data))` — es decir, igual que producción (bytes en RAM, ADR-0002).

Resultados:

- **`simple.pdf`** (3 líneas a 24/16/12pt): se recuperan los 3 tamaños de fuente con Y distinto por línea → base para mapear headers por tamaño.
- **`table.pdf`** (grilla 2×3): el texto queda con **X consistente por columna** (59.5 / 201.3 / 343.0 pt) e **Y consistente por fila** (742.5 / 708.5 / 674.5 pt), más 9 rectángulos de celda → base para detectar tablas por alineación.
- **`scanned.pdf`** (solo dibujos): 0 textos, 1 rectángulo → permite detectar `ErrNoExtractableText`.
- **`corrupt.pdf`** (truncado): `NewReader` devuelve `not a PDF file: missing %%EOF` → error limpio, sin crash.

**Hallazgo importante para el extractor:** `Page.Content().Text` devuelve **un `Text` por carácter**, todos con el mismo `X/Y` para un mismo `Tj` (y `W=0`). Para reconstruir palabras/líneas hay que agrupar por `(X, Y)` preservando el orden, o usar el helper `GetStyledTexts()`/`GetTextByRow()` que ya lo hace.

## Decisión

Usar **`github.com/ledongthuc/pdf`** para la extracción de estructura.

Motivos:

- Único candidato que cumple licencia (BSD-3-Clause) **y** expone fuente + coordenadas sin cgo.
- Fork de `rsc.io/pdf` con mayor tracción (625★, commits recientes, `go.mod` válido).
- API mínima y directa: `NewReader(io.ReaderAt, size)` → `Reader.Page(n).Content()`.

## Consecuencias

**Positivas:**

- Sin cgo: build e imagen Docker triviales, sin toolchain nativo.
- Licencia BSD permite **vendoring/fork** si el mantenimiento se estanca.
- Mapeo directo a nuestro `Block` (`FontSize`, `X`, `Y`, `W`, `Page`).

**Negativas / a tener en cuenta:**

- `rsc.io/pdf` está archivado; `ledongthuc/pdf` es mantenido por la comunidad, no por un vendor. Mitigación: BSD permite forkear.
- El soporte de PDFs **encriptados** es débil (el driver expone `NewReaderEncrypted` con callback de password; el caso "con contraseña" se trata como error de dominio).
- El parser puede hacer **panic** con PDFs muy malformados (aunque `corrupt.pdf` devolvió error limpio). Obligatorio envolver la extracción en `recover()` (ya previsto en el plan, Fase 2).
- Hay que reconstruir palabras/líneas desde los `Text` por carácter (o usar `GetStyledTexts()`).

## Alternativas descartadas

- **unidoc/unipdf**: potente, pero licencia comercial/AGPL con API key — riesgo legal para la organización.
- **pdfcpu**: excelente para manipular/validar PDFs, pero no expone posición/fuente por glifo — insuficiente para estructura.
- **go-pdfium / go-fitz**: más robustos, pero cgo (y MuPDF es AGPL) — complejidad de build que no se justifica.
- **pdflib**: comercial y wrapper C — descartado por licencia y cgo.

## Reproducir

```bash
cd research
go run . gen      # genera los fixtures en ../testdata
go run .          # vuelca estructura de cada fixture con ledongthuc/pdf
```
