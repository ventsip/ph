package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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

func pgbalance(ph *engine.ProcessHunter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.MarshalIndent(ph.GetLatestPGroupsBalance(), "", "    ")
		fmt.Fprintf(w, "%s", b)
	}
}

func pbalance(ph *engine.ProcessHunter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.MarshalIndent(ph.GetLatestProcessesBalance(), "", "    ")
		fmt.Fprintf(w, "%s", b)
	}
}

// Serve serves web interface for ph
func Serve(ph *engine.ProcessHunter) {
	mux := http.NewServeMux() // avoid using DefaultServeMux

	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("/", fs)

	mux.HandleFunc("/config", config(ph))
	mux.HandleFunc("/pgbalance", pgbalance(ph))
	mux.HandleFunc("/pbalance", pbalance(ph))

	log.Fatal(http.ListenAndServe(port, mux))
}