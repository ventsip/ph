package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"bitbucket.org/ventsip/ph/engine"
)

const port = ":8080"

func config(ph *engine.ProcessHunter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.MarshalIndent(ph.GetLimits(), "", "    ")
		fmt.Fprintf(w, "%s", b)
	}
}

func groupbalance(ph *engine.ProcessHunter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.MarshalIndent(ph.GetLatestPGroupsBalance(), "", "    ")
		fmt.Fprintf(w, "%s", b)
	}
}

func processbalance(ph *engine.ProcessHunter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.MarshalIndent(ph.GetLatestProcessesBalance(), "", "    ")
		fmt.Fprintf(w, "%s", b)
	}
}

func version(ver string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, ver)
	}
}

// Serve serves web interface for ph
func Serve(ctx context.Context, wg *sync.WaitGroup, ph *engine.ProcessHunter, ver string) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()

	mux := http.NewServeMux() // avoid using DefaultServeMux

	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/", fs)
	mux.HandleFunc("/version", version(ver))
	mux.HandleFunc("/config", config(ph))
	mux.HandleFunc("/groupbalance", groupbalance(ph))
	mux.HandleFunc("/processbalance", processbalance(ph))

	s := http.Server{Addr: port, Handler: mux}

	log.Println("starting service")
	go s.ListenAndServe()

	<-ctx.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s.Shutdown(ctx)
	log.Println("web service shut down.")
}
