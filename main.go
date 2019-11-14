package main

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	ps "github.com/mitchellh/go-ps"
)

// scheduler runs the work function periodically (every period seconds)
func scheduler(ctx context.Context, wg *sync.WaitGroup, period time.Duration, work func(context.Context) error) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	err := work(ctx)
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			work(ctx)
		}
	}
}

func getRunningProcess(ctx context.Context) (processes []string, err error) {
	pss, err := ps.Processes()

	if err != nil {
		log.Println(err)
		return nil, err
	}

	for _, p := range pss {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			processes = append(processes, p.Executable())
		}
	}

	return
}

func checkProcesses(ctx context.Context) error {

	pss, err := getRunningProcess(ctx)

	if err != nil {
		return err
	}

	sort.Strings(pss)

	for _, p := range pss {
		fmt.Println(p)
	}

	return nil
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())

	// period defines how often the proccess list is checked
	const period = time.Minute * 3

	var wg sync.WaitGroup
	wg.Add(1)
	go scheduler(ctx, &wg, period, checkProcesses)

	time.Sleep(time.Second * 3)
	cancel()

	wg.Wait()
}
