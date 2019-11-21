package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("Starting")

	term := make(chan os.Signal, 2)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	timer := time.NewTimer(10 * time.Second)

	for {
		select {
		case <-term:
			log.Println("SIGTERM received. Exiting")
			return
		case <-timer.C:
			log.Println("Sleeping")
		}
	}
}
