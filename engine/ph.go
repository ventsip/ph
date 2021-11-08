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

// time format used in downtime specs
const dtTimeFormat = "15:04"

const noLimit = time.Hour * 10000

// DayLimits maps days to time limit
// The key (days) can be
// - "*" (meaing 'any day of the week')
// - space separated string of three-letter abbreviations of the days of week, i.e. Mon Tue Wed Thu Fri Sat Sun
// - a concreate date in the format YYYY-MM-DD
// - space separated list of dates
// - a combination of all of the above
type DayLimits map[string]time.Duration

// Downtime maps days to a list of downtime periods
// See DayLimits for the meaning of the key of this map
// The values (downtime periods) are strings like this:
// "12:00..12:30" - for a 30 minutes downtime
// "..10:00" - downtime up until 10:00 in the morning
// "18:00.." - downtime after 6:00PM
type Downtime map[string][]string

// ProcessGroupDayLimit specifies day time limit DL and downtime periods DT
// for one or more processes in PG
type ProcessGroupDayLimit struct {
	PG []string  `json:"processes"`
	DL DayLimits `json:"limits"`
	DT Downtime  `json:"downtime"`
}

// prettyDuration only purpose is to override MarshalJSON to present time.Duration in more human friendly format
type prettyDuration struct {
	time.Duration
}

// ProcessGroupDayBalance describes day limits and monitored properties of a process group PG
type ProcessGroupDayBalance struct {
	PG        []string       `json:"processes"`
	Limit     prettyDuration `json:"limit"`
	Balance   prettyDuration `json:"balance"`
	Downtime  []string       `json:"downtime"`
	Blocked   bool           `json:"blocked"`
	TimeStamp string         `json:"timestamp"`
}

// TimeBalance maps process name to running time
type TimeBalance map[string]time.Duration

// dayTimeBalance maps date to process running time
type dayTimeBalance map[string]TimeBalance

// ProcessHunter is monitoring and killing processes that go overtime for particular day
type ProcessHunter struct {
	limitsRWM  sync.RWMutex
	limits     []ProcessGroupDayLimit // configuration
	limitsHash uint32                 // checksum of the loaded configuration (limits)

	balanceRWM  sync.RWMutex
	balance     dayTimeBalance
	checkPeriod time.Duration // how often to check processes
	forceCheck  chan struct{} // channel that forces balance check (outsite of checkPeriod)
	balancePath string        // where balance is periodically stored
	savePeriod  time.Duration // how often to save balance to balancePath

	killer func(pid int) error

	cfgPath string    // path to the config file
	cfgTime time.Time // write time stamp of the cfgPath. populated when config file is loaded

	pgroupsRWM   sync.RWMutex
	pgroups      []ProcessGroupDayBalance // latest balance of monitored process groups
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
		forceCheck:  make(chan struct{}),
		balance:     make(dayTimeBalance),
		balancePath: balancePath,
		savePeriod:  savePeriod,
		killer:      killer,
		cfgPath:     cfgPath,
	}
}

// GetLimits returns current day limits (which are normally loaded from a config file) and its hash
func (ph *ProcessHunter) GetLimits() ([]ProcessGroupDayLimit, uint32) {
	ph.limitsRWM.RLock()
	defer ph.limitsRWM.RUnlock()

	return ph.limits, ph.limitsHash
}

// GetLatestPGroupsBalance returns pgroups
func (ph *ProcessHunter) GetLatestPGroupsBalance() []ProcessGroupDayBalance {
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

// getActiveSpec iterates over specs,
// which is an array of keys that are used in DayLimits and Downtime structures.
// It returns the sp (an element of the specs array) and a boolean if such was found
// based on the current date and day of week
// it prioritizes more concrete, to more generic specifications, in order:
// - specific day, e.g. "wed"
// - a day from a list: "mon wed fri"
// - any day "*"
func getActiveSpec(dt string, wd string, specs []string) (sp string, found bool) {
	found = false

	dateInList := false
	dayInList := false
	dayMatch := false
	for _, k := range specs {
		if k == dt { // date match - end of search
			sp = k
			found = true
			break
		}
		if strings.Contains(k, dt) { // date in list
			sp = k
			found = true
			dateInList = true
		}
		if !dateInList {
			if k == wd { // day of week match
				sp = k
				found = true
				dayMatch = true
			}
			if !dayMatch {
				if strings.Contains(k, wd) { // day in list
					sp = k
					found = true
					dayInList = true
				}
				if !dayInList {
					if k == "*" {
						sp = k
						found = true
					}
				}
			}
		}
	}
	return
}

// evalDayLimit returns the day time limit,
// based on the current date dt and week day wd, and the provided DayLimit spec dl
// See getActiveSpec to understand how a particular DayLimit is selected from dl based on dt and dl
func evalDayLimit(dt string, wd string, dl DayLimits) (l time.Duration) {
	l = noLimit // effectively - no limit

	specs := make([]string, len(dl))
	i := 0
	for k := range dl {
		specs[i] = k
		i++
	}

	spec, found := getActiveSpec(dt, wd, specs)

	if found {
		l = dl[spec]
	}

	return
}

// isOvertime evaluates whether the balance exceeds the active day limit,
// based on the current date dt and week day wd, and the provided DayLimits spec dl
// isOvertime returns overtime - the result of the evaluation and limit - the active day limit
// See getActiveSpec to understand how a particular limit is selected from dl based on dt and dl
func isOvertime(balance time.Duration, dt string, wd string, dl DayLimits) (overtime bool, limit time.Duration) {
	limit = evalDayLimit(dt, wd, dl)
	overtime = balance > limit
	return
}

// isBlocked evaluates whether now is within bo period,
// based on the current date dt and week day wd, and the provided Downtime spec bo
// isBlocked returns blocked - the result of the evaluation and boSpec - the active downtime specification
// See getActiveSpec to understand how a particular boSpec is selected from bo based on dt and dl
func isBlocked(now time.Time, dt string, wd string, dnt Downtime) (blocked bool, boSpec []string) {

	specs := make([]string, len(dnt))
	i := 0
	for k := range dnt {
		specs[i] = k
		i++
	}

	spec, found := getActiveSpec(dt, wd, specs)

	if found {
		boSpec = dnt[spec]

		// strip down everything, except HH:MM
		now, _ = time.Parse(dtTimeFormat, now.Format(dtTimeFormat))

		for _, period := range boSpec {

			separator := strings.Index(period, "..")

			intersect := true

			if separator > 0 {
				t, err := time.Parse(dtTimeFormat, period[0:separator])
				if err == nil {
					if t.After(now) {
						intersect = false
					}
				}
			}
			if len(period) > 2 {
				t, err := time.Parse(dtTimeFormat, period[separator+2:])
				if err == nil {
					if t.Before(now) {
						intersect = false
					}
				}
			}
			if intersect {
				blocked = true
				return
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

	ph.pgroups = make([]ProcessGroupDayBalance, len(ph.limits))
	ph.processes = make(TimeBalance)

	d := ph.balance[date]
	for il, pgdl := range ph.limits { // iterate all processes day limits
		bg := time.Duration(0)
		for _, p := range pgdl.PG { // iterate all processes in the process group
			bg = bg + d[p]
			ph.processes[p] = d[p].Round(time.Second)
		}

		isOvertime, l := isOvertime(bg, date, weekDay, pgdl.DL)
		now := time.Now()
		isBlocked, dnt := isBlocked(now, date, weekDay, pgdl.DT)

		ph.pgroups[il] = ProcessGroupDayBalance{
			PG:        pgdl.PG,
			Limit:     prettyDuration{l},
			Balance:   prettyDuration{bg.Round(time.Second)},
			Downtime:  dnt,
			Blocked:   isBlocked,
			TimeStamp: now.Format(dtTimeFormat),
		}

		// if overtime or blocked - kill the prcesses
		if isOvertime || isBlocked {
			log.Println(pgdl.PG, ":", bg, "/", l)
			for _, p := range pgdl.PG { // iterate all processes in the process group
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
			log.Println(pgdl.PG, "remaining:", l-bg)
		}
	}

	// 3. Save time balance
	// ---------------
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
	scheduler(ctx, wg, ph.checkPeriod, ph.forceCheck, ph.checkProcesses)
}

// scheduler runs the work function periodically (every period seconds)
// ctx is used to exit the function, wg is the wait group that tracks when the function ends
// force is a channel that forces work to run even before the period expires
func scheduler(ctx context.Context, wg *sync.WaitGroup, period time.Duration, force <-chan struct{}, work func(context.Context, time.Duration) error) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
	}()

	ticker := time.NewTicker(period)
	defer ticker.Stop()

	t := time.Now()

	for {
		dt := time.Since(t)
		if dt > period*2 && period >= time.Minute {
			log.Println("Unusually long duration", dt, "between two process checks (for period", period, "). Have computer woke up from sleep?")
			dt = 0
		}
		work(ctx, dt)
		t = time.Now()

		select {
		case <-ctx.Done():
			return
		case <-force:
		case <-ticker.C:
		}
	}
}

// add adds t to the balance of the process proc for the day
func (dr *dayTimeBalance) add(day string, proc string, t time.Duration) {
	if _, dOk := (*dr)[day]; !dOk {
		(*dr)[day] = make(TimeBalance)
	}

	(*dr)[day][proc] = (*dr)[day][proc] + t
}

// toText returns string representation of the date of t
func toText(t time.Time) string {
	return t.Format("2006-01-02")
}
