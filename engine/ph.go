package engine

import (
	"context"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/go-ps"
)

const noLimit = time.Hour * 10000

// DailyLimits maps days of the week to time limit
// The key can be "*" (meaing 'any day of the week') or space separated string of
// three-letter abbreviations of the days of week, i.e.
// Mon Tue Wed Thu Fri Sat Sun
type DailyLimits map[string]time.Duration

// ProcessGroupDailyLimit specifies daily time limit DL
// for one or more processes in PG
type ProcessGroupDailyLimit struct {
	PG []string    `json:"processes"`
	DL DailyLimits `json:"limits"`
}

// prettyDuration only purpose is to override MarshalJSON to present time.Duration in more human friendly format
type prettyDuration struct {
	time.Duration
}

// ProcessGroupDailyBalance describes daily limit L and balance B of process group PG
type ProcessGroupDailyBalance struct {
	PG []string       `json:"processes"`
	L  prettyDuration `json:"limit"`
	B  prettyDuration `json:"balance"`
}

// TimeBalance maps process name to running time
type TimeBalance map[string]time.Duration

// dailyTimeBalance maps date to process running time
type dailyTimeBalance map[string]TimeBalance

// ProcessHunter is monitoring and killing processes that go overtime for particular day
type ProcessHunter struct {
	limitsRWM  sync.RWMutex
	limits     []ProcessGroupDailyLimit // configuration
	limitsHash uint32                   // checksum of the loaded configuration (limits)

	balanceRWM  sync.RWMutex
	balance     dailyTimeBalance
	checkPeriod time.Duration // how often to check processes
	balancePath string        // where balance is periodically stored
	savePeriod  time.Duration // how often to save balance to balancePath

	killer func(pid int) error

	cfgPath string    // path to the config file
	cfgTime time.Time // write time stamp of the cfgPath. populated when config file is loaded

	pgroupsRWM   sync.RWMutex
	pgroups      []ProcessGroupDailyBalance // latest balance of monitored process groups
	processesRWM sync.RWMutex
	processes    TimeBalance // latest balance of monitored processes
}

// NewProcessHunter initializes and returns a new ProcessHunter
func NewProcessHunter(
	checkPeriod time.Duration,
	balancePath string,
	savePeriod time.Duration,
	killer func(int) error,
	cfgPath string) *ProcessHunter {
	return &ProcessHunter{
		checkPeriod: checkPeriod,
		balance:     make(dailyTimeBalance),
		balancePath: balancePath,
		savePeriod:  savePeriod,
		killer:      killer,
		cfgPath:     cfgPath,
	}
}

// GetLimits returns current daily limits (which are normally loaded from a config file) and its hash
func (ph *ProcessHunter) GetLimits() ([]ProcessGroupDailyLimit, uint32) {
	ph.limitsRWM.RLock()
	defer ph.limitsRWM.RUnlock()

	return ph.limits, ph.limitsHash
}

// GetLatestPGroupsBalance returns pgroups
func (ph *ProcessHunter) GetLatestPGroupsBalance() []ProcessGroupDailyBalance {
	ph.pgroupsRWM.RLock()
	defer ph.pgroupsRWM.RUnlock()

	return ph.pgroups
}

// GetLatestProcessesBalance returns processes
func (ph *ProcessHunter) GetLatestProcessesBalance() TimeBalance {
	ph.processesRWM.RLock()
	defer ph.processesRWM.RUnlock()

	return ph.processes
}

// savePeriod is when the balance was last saved
var lastSaved = time.Now()

var weekDays = [...]string{
	"sun",
	"mon",
	"tue",
	"wed",
	"thu",
	"fri",
	"sat",
}

// evalDailyLimit returns the daily time limit, parsing dl map
// prioritizing more concrete, to more generic specifications, in order:
// - specific day, e.g. "wed"
// - a day from a list: "mon wed fri"
// - any day "*"
func evalDailyLimit(dt string, wd string, dl DailyLimits) (l time.Duration) {
	l = noLimit // effectively - no limit
	dateInList := false
	dayInList := false
	dayMatch := false
	for k, v := range dl {
		if k == dt { // date match - end of search
			l = v
			break
		}
		if strings.Contains(k, dt) { // date in list
			l = v
			dateInList = true
		}
		if !dateInList {
			if k == wd { // day of week match
				l = v
				dayMatch = true
			}
			if !dayMatch {
				if strings.Contains(k, wd) { // day in list
					l = v
					dayInList = true
				}
				if !dayInList {
					if k == "*" {
						l = v
					}
				}
			}
		}
	}
	return
}

// reloadConfigIfNeeded reloads the config file if it has changed
// since last config load
func (ph *ProcessHunter) reloadConfigIfNeeded() (bool, error) {
	if ph.cfgPath == "" {
		return false, nil
	}

	file, err := os.Stat(ph.cfgPath)
	if err != nil {
		return false, err
	}

	if file.ModTime() != ph.cfgTime {
		return true, ph.LoadConfig()
	}

	return false, nil
}

// checkProcesses updates processes time balance (adding dt), checks for overtime and kills processes
func (ph *ProcessHunter) checkProcesses(ctx context.Context, dt time.Duration) error {

	// 0. reload config file, if necessary
	// ---------------
	b, err := ph.reloadConfigIfNeeded()
	if err != nil {
		log.Println("error attempting to reload config:", err)
	}

	if b {
		log.Println("config reloaded:")
		log.Println(ph.GetLimits())
	}

	// check if context is cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// 1. get all processes and update their time balance for the day
	// ---------------

	pss, err := ps.Processes()

	if err != nil {
		log.Println(err)
		return err
	}

	now := time.Now()
	date := toText(now)
	weekDay := weekDays[now.Weekday()]

	ph.balanceRWM.Lock()
	defer ph.balanceRWM.Unlock()

	for _, p := range pss {
		ph.balance.add(date, p.Executable(), dt)
	}

	// 2. check which processes are overtime and kill them
	// ---------------
	ph.limitsRWM.RLock()
	defer ph.limitsRWM.RUnlock()
	ph.pgroupsRWM.Lock()
	defer ph.pgroupsRWM.Unlock()
	ph.processesRWM.Lock()
	defer ph.processesRWM.Unlock()

	ph.pgroups = make([]ProcessGroupDailyBalance, len(ph.limits))
	ph.processes = make(TimeBalance)

	d := ph.balance[date]
	for il, pdl := range ph.limits { // iterate all processes daily limits
		bg := time.Duration(0)
		for _, p := range pdl.PG { // iterate all processes in the process group
			bg = bg + d[p]
			ph.processes[p] = d[p].Round(time.Second)
		}

		l := evalDailyLimit(date, weekDay, pdl.DL)

		ph.pgroups[il] = ProcessGroupDailyBalance{
			PG: pdl.PG,
			L:  prettyDuration{l},
			B:  prettyDuration{bg.Round(time.Second)},
		}

		if bg > l {
			log.Println(pdl.PG, ":", bg, "/", l)
			for _, p := range pdl.PG { // iterate all processes in the process group
				if d[p] > 0 {
					log.Println(p, ":", d[p])
					for _, a := range pss { // iterate all running processes
						if a.Executable() == p {
							// check if context is cancelled before attempting to kill
							select {
							case <-ctx.Done():
								return ctx.Err()
							default:
								log.Println("killing", a.Pid())
								err := ph.killer(a.Pid())
								if err != nil {
									log.Println("error killing", a.Pid(), ":", err.Error())
								}
							}
						}
					}
				}
			}
		} else {
			log.Println(pdl.PG, "remaining:", l-bg)
		}
	}

	if (lastSaved.Add(ph.savePeriod)).Before(time.Now()) {
		if ph.balancePath != "" {
			log.Println("saving balance", ph.balancePath)
			err := ph.saveBalance()

			if err != nil {
				log.Println("error saving balance to", ph.balancePath, ":", err)
			} else {
				lastSaved = time.Now()
			}
		}
	}

	return nil
}

// Run is a goroutine that periodically checks running processes
func (ph *ProcessHunter) Run(ctx context.Context, wg *sync.WaitGroup) {
	scheduler(ctx, wg, ph.checkPeriod, ph.checkProcesses)
}

// scheduler runs the work function periodically (every period seconds)
func scheduler(ctx context.Context, wg *sync.WaitGroup, period time.Duration, work func(context.Context, time.Duration) error) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	t := time.Now()
	err := work(ctx, 0) // don't add anything to process balance on the first call
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dt := time.Now().Sub(t)
			if dt > period*2 && period >= time.Minute {
				log.Println("Unusually long duration", dt, "between two process checks (for period", period, "). Have computer woke up from sleep?")
				dt = 0
			}
			work(ctx, dt)
			t = time.Now()
		}
	}
}

// add adds t to the balance of the process proc for the day
func (dr *dailyTimeBalance) add(day string, proc string, t time.Duration) {
	if _, dOk := (*dr)[day]; !dOk {
		(*dr)[day] = make(TimeBalance)
	}

	(*dr)[day][proc] = (*dr)[day][proc] + t
}

// toText returns string representation of the date of t
func toText(t time.Time) string {
	return t.Format("2006-01-02")
}
