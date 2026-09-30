// Package checker verifica la salud de endpoints HTTP de forma concurrente.
//
// El tipo Checker encapsula la configuración (cliente HTTP, timeout y grado de
// concurrencia) y expone Check.
package checker

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"
)

type Status string

const (
	StatusOK      Status = "OK"
	StatusFail    Status = "FAIL"
	StatusTimeout Status = "TIMEOUT"
)

type Result struct {
	URL     string
	Code    int
	Latency time.Duration
	Status  Status
	Err     error
}

type Checker struct {
	Client      *http.Client
	Timeout     time.Duration
	Concurrency int
}

// Crea un Checker con un http.Client por defecto. La concurrencia se acota a
// un mínimo de 1 para evitar un semáforo de tamaño cero (que bloquearía).
func New(timeout time.Duration, concurrency int) *Checker {
	if concurrency < 1 {
		concurrency = 1
	}
	return &Checker{
		Client:      &http.Client{},
		Timeout:     timeout,
		Concurrency: concurrency,
	}
}

// Verifica todas las urls en paralelo y devuelve los resultados en el mismo
// orden que la entrada.
//
// Concurrencia:
//   - Se crea una goroutine por URL; sync.WaitGroup espera a que todas terminen.
//   - Un "semáforo" (canal con buffer del tamaño de Concurrency) limita cuántas
//     verificaciones corren a la vez, evitando abrir miles de conexiones de golpe.
//   - Cada goroutine escribe su Result en su propia posición del slice, así no
//     hay escrituras concurrentes al mismo índice.
func (c *Checker) Check(ctx context.Context, urls []string) []Result {
	results := make([]Result, len(urls))

	var wg sync.WaitGroup
	sem := make(chan struct{}, c.Concurrency) // semáforo para acotar la paralelización

	for i, u := range urls {
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()

			sem <- struct{}{}        // adquiere un cupo (bloquea si ya hay Concurrency activas)
			defer func() { <-sem }() // libera el cupo al terminar

			results[idx] = c.checkOne(ctx, url)
		}(i, u)
	}

	wg.Wait()
	return results
}

// checkOne verifica un único endpoint.
//
// Context y timeouts:
//   - Se deriva un context con timeout por endpoint mediante context.WithTimeout
//     sobre el context padre. Si el servidor no responde a tiempo, el context se
//     cancela y la request aborta con un error de deadline exceeded.
//   - defer cancel() libera los recursos del context aunque la request termine
//     antes del timeout (evita fugas de recursos internos del context).
func (c *Checker) checkOne(parent context.Context, url string) Result {
	res := Result{URL: url}

	ctx, cancel := context.WithTimeout(parent, c.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		// URL malformada u otro error al construir la request
		res.Status = StatusFail
		res.Err = err
		return res
	}

	start := time.Now()
	resp, err := c.Client.Do(req)
	res.Latency = time.Since(start)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			res.Status = StatusTimeout
		} else {
			res.Status = StatusFail
		}
		res.Err = err
		return res
	}
	defer resp.Body.Close()

	res.Code = resp.StatusCode

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		res.Status = StatusOK
	} else {
		res.Status = StatusFail
	}
	return res
}
