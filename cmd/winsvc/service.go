// +build windows

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bitbucket.org/ventsip/ph/engine"
	"bitbucket.org/ventsip/ph/server"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

var elog debug.Log

type myservice struct{}

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	log.Println(version)
	defer log.Println("exiting.")

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	// change local directory
	exe, err := os.Executable()
	if err != nil {
		log.Println("cannot get executable", err)
	}
	wd, err := filepath.Abs(filepath.Dir(exe))
	if err != nil {
		log.Println("cannot find absolute file path:", err)
	}
	log.Println("changing working directory to:", wd)
	if err := os.Chdir(wd); err != nil {
		log.Println("error", err, "changing local directory", wd)
	}

	// period defines how often the proccess list is checked
	const checkPeriod = time.Minute * 3
	const savePeriod = time.Minute * 5
	const cfgFile = "cfg.json"
	const balanceFile = "balance.json"

	ph := engine.NewProcessHunter(checkPeriod, balanceFile, savePeriod, engine.Kill, cfgFile)

	log.Println("config:", cfgFile)
	if err := ph.LoadConfig(); err != nil {
		log.Println("error loading config file:", err)
		// continue - the cfg file may be reloaded later
	}

	log.Println(ph.GetLimits())

	if err := ph.LoadBalance(); err != nil {
		log.Println("error loading balance file:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go ph.Run(ctx, &wg)
	wg.Add(1)
	go server.Serve(ctx, &wg, ph, version)

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus

			case svc.Stop, svc.Shutdown:
				// golang.org/x/sys/windows/svc.TestExample is verifying this output.
				testOutput := strings.Join(args, "-")
				testOutput += fmt.Sprintf("-%d", c.Context)
				elog.Info(1, testOutput)

				changes <- svc.Status{State: svc.StopPending}

				cancel()
				wg.Wait()

				if err := ph.SaveBalance(); err != nil {
					log.Println("error saving balance:", err)
				}

				changes <- svc.Status{State: svc.Stopped}
				return

			case svc.Pause:

				cancel()
				wg.Wait()
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}

			case svc.Continue:

				ctx, cancel = context.WithCancel(context.Background())
				defer cancel()
				wg.Add(1)
				go ph.Run(ctx, &wg)
				wg.Add(2)
				go server.Serve(ctx, &wg, ph)
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
}

func runService(name string, isDebug bool) {
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &myservice{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
}
