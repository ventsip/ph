package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ventsip/ph/engine"
)

// some test configuration
// don't use tabs - only spaces; otherwise the some string comparisons may fail
const cfg = `[
    {
        "processes": [
            "non.existing.process.name.with"
        ],
        "limits": {
            "mon": "168h0m0s"
        },
        "downtime": null
    }
]`

func TestGetConfigHandler(t *testing.T) {
	ph := engine.NewProcessHunter(time.Hour, "", time.Hour, nil, "")
	err := ph.SetConfig([]byte(cfg))
	if err != nil {
		t.Fatal("Could not set config:", cfg)
	}

	h := http.Handler(config(ph))
	rec := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/config", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			rec.Code, http.StatusOK)
	}

	if rec.Body.String() != cfg {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rec.Body.String(), cfg)
	}

	if ctype := rec.Header().Get("Content-Type"); ctype != "application/json; charset=utf-8" {
		t.Errorf("content type header does not match: got %v want %v",
			ctype, "application/json; charset=utf-8")
	}
}

func TestPutConfigHandler(t *testing.T) {
	ph := engine.NewProcessHunter(time.Hour, "", time.Hour, nil, "")

	h := http.Handler(config(ph))
	rec := httptest.NewRecorder()
	r, err := http.NewRequest("PUT", "/config", strings.NewReader(cfg))
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rec, r)
	if rec.Code != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v",
			rec.Code, http.StatusCreated)
	}

	l, _ := ph.GetLimits()
	b, err := json.MarshalIndent(l, "", "    ")
	if err != nil {
		t.Error("Cannot marshal config to JSON?")
	}
	if string(b) != cfg {
		t.Error("Config was not set correctly")
	}
}

func TestAuthPutHandler(t *testing.T) {
	called := false
	h := http.Handler(authPut(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})))
	rec := httptest.NewRecorder()
	r, err := http.NewRequest("PUT", "not relevant", nil)
	if err != nil {
		t.Fatal(err)
	}

	h.ServeHTTP(rec, r)

	if called {
		t.Error("Successful call without providing credentials")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v",
			rec.Code, http.StatusUnauthorized)
	}

	if rec.Header().Get("WWW-Authenticate") != `Basic realm="Configuration"` {
		t.Error(`Response header does not include "WWW-Authenticate" challenge`)
	}
}
func TestAuthPutBadCredentials(t *testing.T) {
	called := false
	h := http.Handler(authPut(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})))
	rec := httptest.NewRecorder()
	r, err := http.NewRequest("PUT", "not relevant", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.SetBasicAuth("wrong user", "wrong password")

	h.ServeHTTP(rec, r)

	if called {
		t.Error("Successful call with wrong credentials")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v", rec.Code, http.StatusUnauthorized)
	}

	if rec.Header().Get("WWW-Authenticate") != `Basic realm="Configuration"` {
		t.Error(`Response header does not include "WWW-Authenticate" challenge`)
	}
}

func TestAuthPutGoodCredentials(t *testing.T) {
	called := false
	h := http.Handler(authPut(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})))
	r, err := http.NewRequest("PUT", "not relevant", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.SetBasicAuth("time", "k33p3rs")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rec.Code, http.StatusOK)
	}

	if !called {
		t.Error("Rejected with right credentials")
	}
}

func TestGetConfig(t *testing.T) {
	ph := engine.NewProcessHunter(time.Second, "", time.Hour, nil, "")
	err := ph.SetConfig([]byte(cfg))
	if err != nil {
		t.Fatal("Could not set config:", cfg)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(1)
	go ph.Run(ctx, &wg)
	wg.Add(1)
	go Serve(ctx, &wg, ph, "test")

	r, err := http.Get("http://localhost:8080/config")
	if err != nil {
		t.Fatal("Error calling", "http://localhost:8080/config")
	}

	if r.StatusCode != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", r.StatusCode, http.StatusOK)
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatal("Cannot read response body")
	}
	if string(b) != cfg {
		t.Errorf("handler returned unexpected body: got %v want %v",
			string(b), cfg)
	}

	if ctype := r.Header.Get("Content-Type"); ctype != "application/json; charset=utf-8" {
		t.Errorf("content type header does not match: got %v want %v", ctype, "application/json; charset=utf-8")
	}

	cancel()
	wg.Wait()
}

func quickTestGetJSON(t *testing.T, url string, ctype string) {
	ph := engine.NewProcessHunter(time.Second, "", time.Hour, nil, "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	wg.Add(1)
	go ph.Run(ctx, &wg)
	wg.Add(1)
	go Serve(ctx, &wg, ph, "test")

	r, err := http.Get(url)
	if err != nil {
		t.Fatal("Error calling", url)
	}

	if r.StatusCode != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			r.StatusCode, http.StatusOK)
	}

	if c := r.Header.Get("Content-Type"); c != ctype {
		t.Errorf("content type header does not match: got %v want %v",
			c, ctype)
	}

	cancel()
	wg.Wait()
}
func TestSimpleGetConfig(t *testing.T) {
	quickTestGetJSON(t, "http://localhost:8080/config", "application/json; charset=utf-8")
}

func TestSimpleGetGroupBalance(t *testing.T) {
	quickTestGetJSON(t, "http://localhost:8080/groupbalance", "application/json; charset=utf-8")
}
func TestSimpleGetProcessBalance(t *testing.T) {
	quickTestGetJSON(t, "http://localhost:8080/processbalance", "application/json; charset=utf-8")
}

func TestSimpleGetVersion(t *testing.T) {
	quickTestGetJSON(t, "http://localhost:8080/version", "text/plain; charset=utf-8")
}

func TestSimpleGetBalance(t *testing.T) {
	quickTestGetJSON(t, "http://localhost:8080/balance", "application/json; charset=utf-8")
}
