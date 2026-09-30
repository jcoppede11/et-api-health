package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestChecker(client *http.Client, timeout time.Duration) *Checker {
	return &Checker{Client: client, Timeout: timeout, Concurrency: 5}
}

func TestCheck_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := newTestChecker(srv.Client(), time.Second).checkOne(context.Background(), srv.URL)

	if res.Status != StatusOK {
		t.Errorf("Status = %q, se esperaba %q", res.Status, StatusOK)
	}
	if res.Code != http.StatusOK {
		t.Errorf("Code = %d, se esperaba %d", res.Code, http.StatusOK)
	}
	if res.Err != nil {
		t.Errorf("Err = %v, se esperaba nil", res.Err)
	}
}

func TestCheck_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	res := newTestChecker(srv.Client(), time.Second).checkOne(context.Background(), srv.URL)

	if res.Status != StatusFail {
		t.Errorf("Status = %q, se esperaba %q", res.Status, StatusFail)
	}
	if res.Code != http.StatusInternalServerError {
		t.Errorf("Code = %d, se esperaba %d", res.Code, http.StatusInternalServerError)
	}
}

func TestCheck_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	res := newTestChecker(srv.Client(), 20*time.Millisecond).checkOne(context.Background(), srv.URL)

	if res.Status != StatusTimeout {
		t.Errorf("Status = %q, se esperaba %q", res.Status, StatusTimeout)
	}
	if res.Err == nil {
		t.Error("Err = nil, se esperaba un error de deadline")
	}
}

func TestCheck_NetworkError(t *testing.T) {
	res := newTestChecker(http.DefaultClient, time.Second).checkOne(context.Background(), "http://127.0.0.1:1/")

	if res.Status != StatusFail {
		t.Errorf("Status = %q, se esperaba %q", res.Status, StatusFail)
	}
	if res.Err == nil {
		t.Error("Err = nil, se esperaba un error de red")
	}
}

func TestCheck_BadURL(t *testing.T) {
	res := newTestChecker(http.DefaultClient, time.Second).checkOne(context.Background(), "://no-es-una-url")

	if res.Status != StatusFail {
		t.Errorf("Status = %q, se esperaba %q", res.Status, StatusFail)
	}
	if res.Err == nil {
		t.Error("Err = nil, se esperaba un error de parseo")
	}
}

func TestCheck_ConcurrentOrder(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer fail.Close()

	c := newTestChecker(ok.Client(), time.Second)
	urls := []string{ok.URL, fail.URL}
	results := c.Check(context.Background(), urls)

	if len(results) != 2 {
		t.Fatalf("len(results) = %d, se esperaba 2", len(results))
	}
	if results[0].URL != ok.URL || results[0].Status != StatusOK {
		t.Errorf("results[0] = %+v, se esperaba OK para %q", results[0], ok.URL)
	}
	if results[1].URL != fail.URL || results[1].Status != StatusFail {
		t.Errorf("results[1] = %+v, se esperaba FAIL para %q", results[1], fail.URL)
	}
}

func TestNew_ClampsConcurrency(t *testing.T) {
	if got := New(time.Second, 0).Concurrency; got != 1 {
		t.Errorf("Concurrency = %d, se esperaba 1", got)
	}
	if got := New(time.Second, -5).Concurrency; got != 1 {
		t.Errorf("Concurrency = %d, se esperaba 1", got)
	}
}
