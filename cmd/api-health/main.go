// Command api-health: verifica la salud de múltiples endpoints HTTP de forma
// concurrente. Para cada URL reporta el código de estado HTTP, la latencia y un
// estado resumido (OK / FAIL / TIMEOUT).
//
// Uso:
//
//	api-health https://example.com https://api.example.com/health
//	api-health -f urls.txt -timeout 3s -concurrency 8
//	api-health -url https://example.com -url https://otra.com
//	api-health -json https://example.com | jq
//
// Las URLs pueden entregarse por argumentos posicionales, por el flag repetible
// o por un archivo con una URL por línea.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jcoppede11/api-health/internal/checker"
	"github.com/jcoppede11/api-health/internal/report"
	"github.com/jcoppede11/api-health/internal/source"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// Permite pasar un flag repetido (-url a -url b) y acumular los
// valores en un slice, en lugar de que la última ocurrencia pise a las previas.
type stringSliceFlag []string

func (s *stringSliceFlag) String() string { return strings.Join(*s, ", ") }

func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// Contiene la lógica del comando y devuelve el código de salida del proceso:
//   - 0: todos los endpoints OK
//   - 1: algún endpoint no está sano, o un error de configuración
//   - 2: error al parsear los flags
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("api-health", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var urlFlags stringSliceFlag
	fs.Var(&urlFlags, "url", "URL a verificar (puede repetirse)")
	file := fs.String("f", "", "archivo con una URL por línea (# para comentarios)")
	timeout := fs.Duration("timeout", 5*time.Second, "timeout por endpoint (ej: 3s, 500ms)")
	concurrency := fs.Int("concurrency", 10, "máximo de verificaciones simultáneas")
	asJSON := fs.Bool("json", false, "salida en formato JSON (machine-readable)")

	if err := fs.Parse(args); err != nil {
		return 2 // flag ya imprimió el error/uso en stderr
	}

	// Reune URLs desde las tres fuentes posibles (flag -url, archivo y args).
	urls, err := source.Collect(urlFlags, *file, fs.Args())
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if len(urls) == 0 {
		fmt.Fprintln(stderr, "error: no se indicaron URLs. Usa argumentos, -url o -f.")
		fs.Usage()
		return 1
	}

	c := checker.New(*timeout, *concurrency)
	results := c.Check(context.Background(), urls)

	if *asJSON {
		if err := report.JSON(stdout, results); err != nil {
			fmt.Fprintln(stderr, "error serializando JSON:", err)
			return 1
		}
	} else {
		report.Table(stdout, results)
	}

	// Código de salida distinto de cero si algún endpoint no está sano. Útil
	// para integrarse en pipelines de CI/monitoreo.
	for _, r := range results {
		if r.Status != checker.StatusOK {
			return 1
		}
	}
	return 0
}
