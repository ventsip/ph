package main

import (
	"bitbucket.org/ventsip/ph"
	"context"
	"sync"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	// period defines how often the proccess list is checked
	const period = time.Minute * 3

	var wg sync.WaitGroup
	wg.Add(1)

	ph := ph.NewProcessHunter(nil, time.Second)
	go ph.Run(ctx, &wg)

	time.Sleep(time.Second * 3)
	cancel()

	wg.Wait()
}
