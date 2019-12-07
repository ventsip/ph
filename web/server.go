package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"bitbucket.org/ventsip/ph/engine"
)

func home(ph *engine.ProcessHunter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.MarshalIndent(ph.GetLimits(), "", " ")
		fmt.Fprintf(w, "Config:\n%s", b)
	}
}

// Serve serves web interface for ph
func Serve(ph *engine.ProcessHunter) {
	http.HandleFunc("/", home(ph))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
