package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"bitbucket.org/ventsip/ph/engine"
)

const port = ":8080"

func home(ph *engine.ProcessHunter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, _ := json.MarshalIndent(ph.GetLimits(), "", "    ")
		fmt.Fprintf(w, "Configuration:\n%s\n", b)

		b, _ = json.MarshalIndent(ph.GetLatestPGroupsBalance(), "", "    ")
		fmt.Fprintf(w, "Process Groups:\n%s\n", b)

		b, _ = json.MarshalIndent(ph.GetLatestProcessesBalance(), "", "    ")
		fmt.Fprintf(w, "Monitored Processes:\n%s\n", b)
	}
}

// Serve serves web interface for ph
func Serve(ph *engine.ProcessHunter) {
	http.HandleFunc("/", home(ph))
	log.Fatal(http.ListenAndServe(port, nil))
}
