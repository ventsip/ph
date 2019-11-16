package main

import (
	"context"
	"log"
	"sync"
	"time"

	"bitbucket.org/ventsip/ph"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// period defines how often the proccess list is checked
	const period = time.Second * 3

	var wg sync.WaitGroup
	wg.Add(1)

	l := make(ph.DailyTimeLimit)
	l["Slack"] = time.Second
	l["zsh"] = time.Second
	ph := ph.NewProcessHunter(l, period, func(pid int) error { log.Println("boom", pid); return nil })
	go ph.Run(ctx, &wg)

	time.Sleep(period * 3)
	cancel()
	wg.Wait()
}
