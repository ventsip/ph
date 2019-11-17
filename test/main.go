package main

import (
	"context"
	"sync"
	"time"

	"bitbucket.org/ventsip/ph"
)

func main() {
	// period defines how often the proccess list is checked
	const period = time.Second * 3

	ph := ph.NewProcessHunter(nil, period, ph.Kill)
	ph.LoadConfig("cfg.json")
	ph.LoadBalance("balance.json")

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	{
		go ph.Run(ctx, &wg)
		time.Sleep(period * 5)
		cancel()
	}
	wg.Wait()

	ph.SaveBalance("balance.json")
}
