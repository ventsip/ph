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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go ph.Run(ctx, &wg)
	wg.Wait()

	ph.SaveBalance("../testdata/balance.json")
}
