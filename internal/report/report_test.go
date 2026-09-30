package report

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jcoppede11/api-health/internal/checker"
)

func sampleResults() []checker.Result {
	return []checker.Result{
		{URL: "https://ok.com", Code: 200, Latency: 90 * time.Millisecond, Status: checker.StatusOK},
		{URL: "https://fail.com", Code: 500, Latency: 120 * time.Millisecond, Status: checker.StatusFail},
		{URL: "https://timeout.com", Latency: 3 * time.Second, Status: checker.StatusTimeout, Err: errors.New("context deadline exceeded")},
	}
}

func TestTable(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, sampleResults())
	out := buf.String()

	for _, want := range []string{"URL", "ESTADO", "https://ok.com", "https://fail.com", "https://timeout.com"} {
		if !strings.Contains(out, want) {
			t.Errorf("la salida no contiene %q\n%s", want, out)
		}
	}

	if !strings.Contains(out, "1 OK, 1 FAIL, 1 TIMEOUT (total 3)") {
		t.Errorf("resumen inesperado:\n%s", out)
	}

	if idxFail, idxOK := strings.Index(out, "fail.com"), strings.Index(out, "ok.com"); idxFail > idxOK {
		t.Errorf("FAIL debería aparecer antes que OK en la tabla:\n%s", out)
	}
}

func TestJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, sampleResults()); err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	var got []jsonResult
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("la salida no es JSON válido: %v\n%s", err, buf.String())
	}

	if len(got) != 3 {
		t.Fatalf("len = %d, se esperaba 3", len(got))
	}

	if got[0].URL != "https://ok.com" || got[0].Status != checker.StatusOK || got[0].LatencyMs != 90 {
		t.Errorf("got[0] = %+v", got[0])
	}

	if got[2].Error == "" {
		t.Error("se esperaba el campo error en el resultado de TIMEOUT")
	}

	if strings.Contains(strings.SplitN(buf.String(), "}", 2)[0], "error") {
		t.Errorf("el primer objeto (OK) no debería tener campo error:\n%s", buf.String())
	}
}
