package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ventsip/ph/engine"
)

const port = ":8080"

// config serves configuration as JSON (GET) and applies new configuration (PUT).
func config(ph *engine.ProcessHunter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			l := ph.GetLimits()
			b, err := json.MarshalIndent(l, "", "    ")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				break
			}
			fmt.Fprintf(w, "%s", b)
		case http.MethodPut:
			b, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				break
			}
			err = ph.SetConfig(b)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				break
			}
			http.Error(w, "Configuration saved", http.StatusCreated)
		default:
			http.Error(w, "Not Implemented", http.StatusNotImplemented)
		}
	})
}

// groupBalance serves ph.GetLatestPGroupsBalance() as JSON (GET)
func groupBalance(ph *engine.ProcessHunter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		b, _ := json.MarshalIndent(ph.GetLatestPGroupsBalance(), "", "    ")
		fmt.Fprintf(w, "%s", b)
	})
}

// processBalance serves ph.GetLatestProcessesBalance() as JSON (GET)
func processBalance(ph *engine.ProcessHunter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		b, _ := json.MarshalIndent(ph.GetLatestProcessesBalance(), "", "    ")
		fmt.Fprintf(w, "%s", b)
	})
}

// balanceHistory serves ph.GetBalance() as JSON (GET)
func balanceHistory(ph *engine.ProcessHunter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=120")
		b, _ := json.MarshalIndent(ph.GetBalance(), "", "    ")
		fmt.Fprintf(w, "%s", b)
	})
}

// version serves version
func version(ver string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, ver)
	})
}

// authPut is a middleware that protects PUT method for handler h with basic authentication.
// Expected username and password are hardcoded.
func authPut(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut { // only PUT method is protected
			w.Header().Set("WWW-Authenticate", `Basic realm="Configuration"`)
			u, p, ok := r.BasicAuth()
			if !ok {
				http.Error(w, "Username and password required", http.StatusUnauthorized)
				return
			}
			if !(u == "time" && p == "k33p3rs") {
				http.Error(w, "Incorrect username or password", http.StatusUnauthorized)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

//go:embed webFiles
var webFolder embed.FS

// this is needed, since go:embed does not support embedding files without the directory name
var webFS, _ = fs.Sub(webFolder, "webFiles")

// Serve serves web interface for ph and signals a channel when the server is ready to receive client connections
func Serve(ctx context.Context, wg *sync.WaitGroup, ph *engine.ProcessHunter, ver string, ready chan<- struct{}) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()

	mux := http.NewServeMux() // avoid using DefaultServeMux

	mux.Handle("/", http.FileServer(http.FS(webFS)))
	mux.Handle("/version", version(ver))
	mux.Handle("/config", authPut(config(ph)))
	mux.Handle("/groupbalance", groupBalance(ph))
	mux.Handle("/processbalance", processBalance(ph))
	mux.Handle("/balance", balanceHistory(ph))

	s := http.Server{Addr: port, Handler: mux}

	log.Println("starting service")
	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("Web service listen error:", err)
		}
	}()

	ready <- struct{}{} // signal that the server is ready

	<-ctx.Done()

	log.Println("Web service shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := s.Shutdown(ctx)
	if err != nil {
		log.Println("Web service shut down error:", err)
	} else {
		log.Println("Web service gracefully shut down")
	}
}
