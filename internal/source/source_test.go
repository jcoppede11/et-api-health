package source

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCollect_DedupAndTrim(t *testing.T) {
	got, err := Collect(
		[]string{"https://a.com", "  https://b.com  "},
		"",
		[]string{"https://a.com", "https://c.com", ""},
	)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	want := []string{"https://a.com", "https://b.com", "https://c.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, se esperaba %v", got, want)
	}
}

func TestCollect_WithFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "urls.txt")
	content := "# comentario\nhttps://file-a.com\n\n  https://file-b.com  \n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("no se pudo escribir el archivo de prueba: %v", err)
	}

	got, err := Collect([]string{"https://flag.com"}, path, nil)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	want := []string{"https://flag.com", "https://file-a.com", "https://file-b.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, se esperaba %v", got, want)
	}
}

func TestCollect_FileNotFound(t *testing.T) {
	if _, err := Collect(nil, "/ruta/que/no/existe.txt", nil); err == nil {
		t.Error("se esperaba un error al abrir un archivo inexistente")
	}
}

func TestCollect_Empty(t *testing.T) {
	got, err := Collect(nil, "", nil)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, se esperaba lista vacía", got)
	}
}
