// Package report formatea los resultados de verificación para su salida de dos maneras diferentes:
//
// - como tabla legible para humanos
// - como JSON para consumo por otras herramientas
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/jcoppede11/et-api-health/internal/checker"
)

// Imprime los resultados en una tabla ordenada, por estado y por URL.
func Table(w io.Writer, results []checker.Result) {
	sorted := make([]checker.Result, len(results))
	copy(sorted, results)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Status != sorted[j].Status {
			return statusRank(sorted[i].Status) < statusRank(sorted[j].Status)
		}
		return sorted[i].URL < sorted[j].URL
	})

	// Calcula el ancho de la columna URL para alinear la tabla.
	urlWidth := len("URL")
	for _, r := range sorted {
		if len(r.URL) > urlWidth {
			urlWidth = len(r.URL)
		}
	}

	fmt.Fprintf(w, "%-*s  %-7s  %6s  %10s\n", urlWidth, "URL", "ESTADO", "CÓDIGO", "LATENCIA")
	fmt.Fprintln(w, strings.Repeat("-", urlWidth+2+7+2+6+2+10))

	var ok, fail, timeout int
	for _, r := range sorted {
		code := "-"
		if r.Code > 0 {
			code = fmt.Sprintf("%d", r.Code)
		}
		fmt.Fprintf(w, "%-*s  %-7s  %6s  %10s\n",
			urlWidth, r.URL, r.Status, code, r.Latency.Round(time.Millisecond))

		if r.Err != nil && r.Status != checker.StatusOK {
			fmt.Fprintf(w, "%-*s  └─ %v\n", urlWidth, "", r.Err)
		}

		switch r.Status {
		case checker.StatusOK:
			ok++
		case checker.StatusTimeout:
			timeout++
		default:
			fail++
		}
	}

	fmt.Fprintf(w, "\nResumen: %d OK, %d FAIL, %d TIMEOUT (total %d)\n",
		ok, fail, timeout, len(sorted))
}

// jsonResult es la vista serializable de un checker.Result.
type jsonResult struct {
	URL       string         `json:"url"`
	Status    checker.Status `json:"status"`
	Code      int            `json:"code"`
	LatencyMs int64          `json:"latency_ms"`
	Error     string         `json:"error,omitempty"`
}

// JSON emite los resultados como un array JSON indentado. Ya preparado para
// pipear a otras herramientas que lea de stdin (jq, grep, etc.).
func JSON(w io.Writer, results []checker.Result) error {
	out := make([]jsonResult, len(results))
	for i, r := range results {
		jr := jsonResult{
			URL:       r.URL,
			Status:    r.Status,
			Code:      r.Code,
			LatencyMs: r.Latency.Milliseconds(),
		}
		if r.Err != nil {
			jr.Error = r.Err.Error()
		}
		out[i] = jr
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// Define el orden de impresión: los problemas primero.
func statusRank(s checker.Status) int {
	switch s {
	case checker.StatusFail:
		return 0
	case checker.StatusTimeout:
		return 1
	default:
		return 2
	}
}
