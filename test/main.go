package main

import (
	"context"
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
	l["test_target.exe"] = time.Second

	ph := ph.NewProcessHunter(l, period, ph.Kill)
	go ph.Run(ctx, &wg)

	time.Sleep(period * 5)
	cancel()
	wg.Wait()
}
