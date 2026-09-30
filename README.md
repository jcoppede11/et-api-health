# api-health

[![CI](https://github.com/jcoppede11/et-api-health/actions/workflows/ci.yml/badge.svg)](https://github.com/jcoppede11/et-api-health/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

CLI tool en Go que verifica la salud de múltiples endpoints HTTP de forma
concurrente. Para cada URL reporta **código de estado HTTP**, **latencia** y un
**estado**: `OK`, `FAIL` o `TIMEOUT`.

Pensada para correr desde la terminal o en CI. Cero dependencias externas, solo
la librería estándar de Go.

## Instalación

```bash
# Instalar el binario en $GOBIN / $GOPATH/bin
go install github.com/jcoppede11/api-health/cmd/api-health@latest
```

O compilar desde el repositorio:

```bash
go build -o api-health ./cmd/api-health
```

## Uso

Las URLs se pueden pasar de tres formas (combinables):

```bash
# Argumentos posicionales
api-health https://example.com https://api.example.com/health

# Flag repetible -url
api-health -url https://example.com -url https://otra.com

# Archivo con una URL por línea (# = comentario)
api-health -f urls.txt
```

### Flags

| Flag           | Default | Descripción                                    |
| -------------- | ------- | ---------------------------------------------- |
| `-url`         | —       | URL a verificar (puede repetirse)              |
| `-f`           | —       | archivo con una URL por línea                  |
| `-timeout`     | `5s`    | timeout por endpoint (ej: `3s`, `500ms`)       |
| `-concurrency` | `10`    | máximo de verificaciones simultáneas           |
| `-json`        | `false` | salida en formato JSON (machine-readable)      |

### Ejemplo

```bash
api-health -f urls.txt -timeout 3s -concurrency 8
```

Salida de ejemplo:

```
URL                                        ESTADO   CÓDIGO    LATENCIA
------------------------------------------------------------------------
https://httpstat.us/500                    FAIL        500       412ms
https://no-existe.dominio-invalido.xyz     FAIL          -         2ms
  └─ Get "https://no-existe...": dial tcp: lookup ...: no such host
https://httpstat.us/200?sleep=10000        TIMEOUT       -      3000ms
https://www.google.com                     OK          200        98ms
https://httpstat.us/200                    OK          200       210ms

Resumen: 2 OK, 2 FAIL, 1 TIMEOUT (total 5)
```

### Códigos de salida

| Código | Significado |
| ------ | ----------- |
| `0`    | Todos los endpoints respondieron `OK` (HTTP 2xx). |
| `1`    | Algún endpoint en `FAIL` o `TIMEOUT`, error al leer `-f`, JSON inválido al serializar, o no se indicaron URLs. |
| `2`    | Error al parsear flags (el uso se imprime en stderr). |

En CI basta con ejecutar el comando, si devuelve distinto de `0`, el job falla sin parsear la tabla.

### Salida JSON

Con `-json` la salida es un array JSON. La latencia se expone en milisegundos:

```bash
api-health -json https://example.com https://httpstat.us/500
```

```json
[
  {
    "url": "https://example.com",
    "status": "OK",
    "code": 200,
    "latency_ms": 98
  },
  {
    "url": "https://httpstat.us/500",
    "status": "FAIL",
    "code": 500,
    "latency_ms": 412
  }
]
```

En respuestas con error de red, el objeto puede incluir `"error": "..."` (campo omitido si no aplica).

## Uso en CI (consumidores)

api-health está pensada para **smoke checks** puntuales: post-deploy, cron o un workflow programado. No reemplaza un sistema de monitoreo continuo (ver [Alcance](#alcance)).

### Instalación en el pipeline

```bash
go install github.com/jcoppede11/api-health/cmd/api-health@latest
```

El binario queda en `$GOBIN` o `$GOPATH/bin`; ese directorio debe estar en el `PATH` del job.

### Ejemplo: GitHub Actions

Guardá las URLs en un archivo del repo (por ejemplo `urls.txt`, una URL por línea) o pasalas con `-url`:

```yaml
jobs:
  api-health:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod   # en tu repo; o fijá una versión, ej. '1.25'

      - name: Instalar api-health
        run: go install github.com/jcoppede11/api-health/cmd/api-health@latest

      - name: Verificar endpoints
        run: api-health -f urls.txt -timeout 10s -concurrency 5
```

Si algún endpoint no está sano, el step falla porque el proceso termina con código `1`.

Variantes habituales:

```bash
# URLs en el workflow (sin archivo)
api-health -url https://prod.example.com/health -timeout 5s

# Salida JSON para logs o artefactos
api-health -json -f urls.txt | tee health-report.json
```

Colocá los flags (`-timeout`, `-f`, `-json`, …) **antes** de las URLs posicionales, para que el parser de flags las reconozca correctamente.

### CI de *este* repositorio

El workflow [`.github/workflows/ci.yml`](.github/workflows/ci.yml) valida el **código fuente** (`gofmt`, `go vet`, tests con cobertura y `-race`, build). No es un ejemplo de chequeo de tus APIs. Para eso usá el patrón de arriba en el repo donde desplegás o consumís la tool.

## Decisiones de diseño

Esta herramienta es deliberadamente pequeña. Estas son las decisiones detrás de
su implementación:

- **Una goroutine por URL.** Es el modelo más
  simple y legible para trabajo I/O-bound independiente.
- **Semáforo (canal con buffer) para acotar la concurrencia.** Sin un límite, una
  lista de miles de URLs abriría miles de conexiones a la vez y podría agotar
  descriptores de archivo. El flag `-concurrency` controla cuántas corren juntas.
- **`context.WithTimeout` por endpoint, no un timeout global.** Así un endpoint
  lento no consume el presupuesto de tiempo de los demás y podemos distinguir un
  `TIMEOUT` (deadline del context) de otros errores de red vía
  `errors.Is(err, context.DeadlineExceeded)`.
- **Cada goroutine escribe en su propio índice del slice de resultados.** No hay
  escrituras concurrentes a la misma posición, así que no hace falta un mutex.
  (El CI de este repo corre `go test -race` para verificarlo.)
- **Un solo `http.Client` reutilizado.** Es seguro para uso concurrente y
  reaprovecha conexiones. El timeout real lo impone el context de cada request.
- **Código de salida `1` si algo no está sano.**

### Alcance

api-health hace un chequeo puntual y termina. No es un sistema de monitoreo:
no tiene modo *watch*, histórico, alertas ni dashboard. Para eso usá herramientas
dedicadas como [blackbox_exporter](https://github.com/prometheus/blackbox_exporter)
o [uptime-kuma](https://github.com/louislam/uptime-kuma).
```
.
├── cmd/
│   └── api-health/      # entry point: parseo de flags y wiring (main delgado)
└── internal/
    ├── checker/         # verificación HTTP concurrente (Checker, Result, Status)
    ├── source/          # recolección de URLs (flags, archivo, argumentos)
    └── report/          # formateo de salida (tabla y JSON)
```

Cada paquete tiene una única responsabilidad y sus propios tests. `main` no
contiene lógica de negocio: delega en una función `run(args, stdout, stderr)`
testeable que devuelve el código de salida.

## Desarrollo

Comandos equivalentes a los del workflow de CI localmente:

```bash
gofmt -l .                   # verificar formato (no debe listar nada)
go vet ./...                 # análisis estático
go test -cover ./...         # tests con cobertura (sin cgo)
go build -o api-health ./cmd/api-health
```

Para desarrollo rápido sin cobertura: `go test ./...`

### Detector de carreras (opcional en local, garantizado en CI)

El detector de carreras (`-race`) valida que la concurrencia no tenga data races,
pero requiere `cgo` y un compilador de C (gcc/clang):

```bash
go test -race ./...
```

En Windows, si obtenés `-race requires cgo`, instalá un compilador de C
(TDM-GCC / mingw-w64) y exportá `CGO_ENABLED=1`. No es imprescindible en local:
**el CI corre `go test -race` en cada push/PR sobre Linux** (junto con `go test -cover`), así que el chequeo siempre se hace.

## Licencia

[MIT](LICENSE).
