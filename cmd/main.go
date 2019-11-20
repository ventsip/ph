package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"bitbucket.org/ventsip/ph/lib"
)

func main() {
	log.Println("Starting")

	// period defines how often the proccess list is checked
	const period = time.Second * 180 // three minutes
	const cfgFile = "cfg.json"
	const balanceFile = "balance.json"

	ph := lib.NewProcessHunter(nil, period, lib.Kill)

	log.Println("loading config")
	if err := ph.LoadConfig(cfgFile); err != nil {
		log.Println("error loading config file", err)
		return
	}

	if err := ph.LoadBalance(balanceFile); err != nil {
		log.Println("error loading balance file", err)
	}

	//ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("SIGTERM received")
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go ph.Run(ctx, &wg)
	wg.Wait()

	ph.SaveBalance(balanceFile)

	log.Println("Exiting")
}
