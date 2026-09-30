// Package source reúne las URLs a verificar desde múltiples fuentes (flags
// repetibles, argumentos posicionales y un archivo), aplicando normalización en una
// única lista sin duplicados.
package source

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Junta las URLs de las tres fuentes y elimina duplicados
//
// u: (abreviatura de url): cadena de URL sin normalizar (flags, args o archivo).
func Collect(fromFlags []string, file string, args []string) ([]string, error) {
	var all []string

	all = append(all, fromFlags...)
	all = append(all, args...)

	if file != "" {
		fromFile, err := readFile(file)
		if err != nil {
			return nil, err
		}
		all = append(all, fromFile...)
	}

	seen := make(map[string]struct{}, len(all))
	var out []string

	for _, u := range all {
		u = strings.TrimSpace(u)

		if u == "" {
			continue
		}

		if _, dup := seen[u]; dup {
			continue
		}

		seen[u] = struct{}{}
		out = append(out, u)
	}

	return out, nil
}

// Lee un archivo con una URL por línea. Ignora líneas vacías y las que
// empiezan con '#' (hashtags) considerados comentarios.
//
// f (abreviatura de fileHandler): archivo abierto en modo lectura.
func readFile(path string) ([]string, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir %q: %w", path, err)
	}
	defer f.Close()

	var urls []string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error leyendo %q: %w", path, err)
	}

	return urls, nil
}
