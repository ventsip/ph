package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bitbucket.org/ventsip/ph/engine"
)

const cfg = `[
    {
        "processes": [
            "p"
        ],
        "limits": {
            "mon": "1s"
        }
    }
]`

func TestGetConfigHandler(t *testing.T) {
	ph := engine.NewProcessHunter(time.Second, "", time.Hour, nil, "")

	err := ph.SetConfig([]byte(cfg))
	if err != nil {
		t.Fatal("Could not set config:", cfg)
	}

	r, err := http.NewRequest("GET", "/config", nil)
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h := http.Handler(config(ph))

	h.ServeHTTP(rec, r)

	if status := rec.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if rec.Body.String() != cfg {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rec.Body.String(), cfg)
	}

	if ctype := rec.Header().Get("Content-Type"); ctype != "application/json" {
		t.Errorf("content type header does not match: got %v want %v",
			ctype, "application/json")
	}
}

func TestPutConfigHandler(t *testing.T) {
	ph := engine.NewProcessHunter(time.Second, "", time.Hour, nil, "")

	r, err := http.NewRequest("PUT", "/config", strings.NewReader(cfg))
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h := http.Handler(config(ph))

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

func TestAuthPutChallenge(t *testing.T) {
	r, err := http.NewRequest("PUT", "not relevant", nil)
	if err != nil {
		t.Fatal(err)
	}

	called := false
	rec := httptest.NewRecorder()
	h := http.Handler(authPut(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})))

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
	r, err := http.NewRequest("PUT", "not relevant", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.SetBasicAuth("wrong user", "wrong password")

	called := false
	rec := httptest.NewRecorder()
	h := http.Handler(authPut(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})))

	h.ServeHTTP(rec, r)

	if called {
		t.Error("Successful call with wrong credentials")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("handler returned wrong status code: got %v want %v",
			rec.Code, http.StatusUnauthorized)
	}

	if rec.Header().Get("WWW-Authenticate") != `Basic realm="Configuration"` {
		t.Error(`Response header does not include "WWW-Authenticate" challenge`)
	}
}

func TestAuthPutGoodCredentials(t *testing.T) {
	r, err := http.NewRequest("PUT", "not relevant", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.SetBasicAuth("time", "keeper")

	called := false
	rec := httptest.NewRecorder()
	h := http.Handler(authPut(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})))

	h.ServeHTTP(rec, r)

	if rec.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			rec.Code, http.StatusOK)
	}

	if !called {
		t.Error("Rejected with right credentials")
	}
}
