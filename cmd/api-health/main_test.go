package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRun_Integration(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer fail.Close()

	var stdout, stderr bytes.Buffer
	code := run([]string{"-json", ok.URL, fail.URL}, &stdout, &stderr)

	if code != 1 {
		t.Errorf("exit code = %d, se esperaba 1\nstderr: %s", code, stderr.String())
	}

	var results []struct {
		URL    string `json:"url"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &results); err != nil {
		t.Fatalf("stdout no es JSON válido: %v\n%s", err, stdout.String())
	}
	if len(results) != 2 {
		t.Fatalf("len = %d, se esperaba 2", len(results))
	}
}

func TestRun_AllOK(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ok.Close()

	var stdout, stderr bytes.Buffer
	if code := run([]string{ok.URL}, &stdout, &stderr); code != 0 {
		t.Errorf("exit code = %d, se esperaba 0\nstderr: %s", code, stderr.String())
	}
}

func TestRun_NoURLs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)

	if code != 1 {
		t.Errorf("exit code = %d, se esperaba 1", code)
	}
	if !strings.Contains(stderr.String(), "no se indicaron URLs") {
		t.Errorf("se esperaba mensaje de error por falta de URLs, got: %s", stderr.String())
	}
}

func TestRun_BadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-flag-inexistente"}, &stdout, &stderr); code != 2 {
		t.Errorf("exit code = %d, se esperaba 2", code)
	}
}
