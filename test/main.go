package main

import (
	"context"
	"log"
	"sync"
	"syscall"
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
	l["test_target"] = time.Second

	f := func(pid int, force bool) error {
		log.Println("boom", pid)
		// Kill the process
		if force {
			return syscall.Kill(pid, syscall.SIGKILL)
		}

		return syscall.Kill(pid, syscall.SIGTERM)
	}

	ph := ph.NewProcessHunter(l, period, f)
	go ph.Run(ctx, &wg)

	time.Sleep(period * 5)
	cancel()
	wg.Wait()
}
