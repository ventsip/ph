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
	const html = `
<head>
    <script src="http://www.w3schools.com/lib/w3data.js"></script>
    <meta http-equiv="refresh" content="60">
	<style>
	table, th, td {border: 1px solid black;}
	</style>
</head>

<body>
    <h1>Process Hunter</h1>
    <h2>Configuration</h2>
    <pre w3-include-html="config"></pre>
    <h2>Process Group Balance</h2>
    <pre w3-include-html="pgbalance"></pre>
    <h2>Process Balance</h2>
    <pre w3-include-html="pbalance"></pre>
    <script>
        w3IncludeHTML();
    </script>
</body>`

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "%s", html)
	}
}
func config(ph *engine.ProcessHunter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json")
		b, _ := json.MarshalIndent(ph.GetLimits(), "", "    ")
		fmt.Fprintf(w, "%s", b)
	}
}
func pgbalance(ph *engine.ProcessHunter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json")
		b, _ := json.MarshalIndent(ph.GetLatestPGroupsBalance(), "", "    ")
		fmt.Fprintf(w, "%s", b)
	}
}

func pbalance(ph *engine.ProcessHunter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/json")
		b, _ := json.MarshalIndent(ph.GetLatestProcessesBalance(), "", "    ")
		fmt.Fprintf(w, "%s", b)
	}
}

// Serve serves web interface for ph
func Serve(ph *engine.ProcessHunter) {
	http.HandleFunc("/", home(ph))
	http.HandleFunc("/config", config(ph))
	http.HandleFunc("/pgbalance", pgbalance(ph))
	http.HandleFunc("/pbalance", pbalance(ph))
	log.Fatal(http.ListenAndServe(port, nil))
}
