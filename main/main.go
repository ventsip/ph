package main

import (
	"context"
	"sync"
	"time"

	"bitbucket.org/ventsip/ph/lib"
)

func main() {
	// period defines how often the proccess list is checked
	const period = time.Second * 3

	ph := lib.NewProcessHunter(nil, period, lib.Kill)
	ph.LoadConfig("../testdata/cfg.json")
	ph.LoadBalance("../testdata/balance.json")

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	{
		go ph.Run(ctx, &wg)
		time.Sleep(period * 5)
		cancel()
	}
	wg.Wait()

	ph.SaveBalance("../testdata/balance.json")
}
