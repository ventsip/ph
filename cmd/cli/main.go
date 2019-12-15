package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"bitbucket.org/ventsip/ph/engine"
	"bitbucket.org/ventsip/ph/server"
)

var version = "undefined"

func main() {
	log.Println(version)
	defer log.Println("exiting.")

	// period defines how often the proccess list is checked
	const checkPeriod = time.Minute * 3
	const savePeriod = time.Minute * 5
	const cfgFile = "cfg.json"
	const balanceFile = "balance.json"

	ph := engine.NewProcessHunter(checkPeriod, balanceFile, savePeriod, engine.Kill, cfgFile)

	log.Println("config:", cfgFile)
	if err := ph.LoadConfig(); err != nil {
		log.Println("error loading config file", err)
		return
	}

	log.Println(ph.GetLimits())

	if err := ph.LoadBalance(); err != nil {
		log.Println("error loading balance file", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGABRT)
	go func() {
		sig := <-c
		log.Println("signal", sig, "received")
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go ph.Run(ctx, &wg)
	wg.Add(1)
	go server.Serve(ctx, &wg, ph)
	wg.Wait()

	if err := ph.SaveBalance(); err != nil {
		log.Println("error saving balance", err)
	}
}
