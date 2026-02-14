package engine

import (
	"context"
	"log"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/mitchellh/go-ps"
)

// time format used in downtime specs
const dtTimeFormat = "15:04"

// DayLimits maps days to time limit
// The key (days) can be
// - "*" (meaning 'any day of the week')
// - space separated string of three-letter abbreviations of the days of week, i.e. Mon Tue Wed Thu Fri Sat Sun
// - a concrete date in the format YYYY-MM-DD
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
	PG []string  `json:"processes"` // PG is the list of process names in this group
	DL DayLimits `json:"limits"`    // DL defines the daily time limits for this group
	DT Downtime  `json:"downtime"`  // DT specifies downtime periods when processes are blocked
}

// prettyDuration only purpose is to override MarshalJSON to present time.Duration in more human friendly format
type prettyDuration struct {
	time.Duration
}

// ProcessGroupDayBalance describes day limits and monitored properties of a process group PG
type ProcessGroupDayBalance struct {
	PG           []string       `json:"processes"`      // PG is the list of process names in this group
	Limit        prettyDuration `json:"limit"`          // Limit is the active daily time limit for the group
	LimitDefined bool           `json:"limit_defined"`  // LimitDefined indicates whether a limit is defined for today
	Balance      prettyDuration `json:"balance"`        // Balance is the total time used by the group today
	Downtime     []string       `json:"downtime"`       // Downtime lists the active downtime periods for today
	Blocked      bool           `json:"blocked"`        // Blocked indicates whether the group is currently in downtime
	TimeStamp    string         `json:"timestamp"`      // TimeStamp is when this balance was calculated (HH:MM format)
}

// TimeBalance maps process name to running time
type TimeBalance map[string]time.Duration

// dayTimeBalance maps date to process running time
type dayTimeBalance map[string]TimeBalance

// ProcessHunter is monitoring and killing processes that go overtime, and during downtime
// for particular day
type ProcessHunter struct {
	limitsRWM sync.RWMutex
	limits    []ProcessGroupDayLimit // configuration

	balanceRWM  sync.RWMutex
	balance     dayTimeBalance // balance history
	checkPeriod time.Duration  // how often to check processes
	forceCheck  chan struct{}  // channel that forces balance check (outside of checkPeriod)
	balancePath string         // where balance is periodically stored
	savePeriod  time.Duration  // how often to save balance to balancePath

	killer func(pid int) error

	cfgPath string    // path to the config file
	cfgTime time.Time // write time stamp of the cfgPath. populated when config file is loaded

	pgroupsRWM   sync.RWMutex
	pgroups      []ProcessGroupDayBalance // latest balance of monitored process groups
	processesRWM sync.RWMutex
	processes    TimeBalance // latest balance of monitored processes

	lastSavedRWM sync.RWMutex
	lastSaved    time.Time // when the balance was last saved
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
		lastSaved:   time.Now(),
	}
}

// GetLimits returns current day limits (which are normally loaded from a config file)
func (ph *ProcessHunter) GetLimits() []ProcessGroupDayLimit {
	ph.limitsRWM.RLock()
	defer ph.limitsRWM.RUnlock()

	return ph.limits
}

// GetLatestPGroupsBalance returns the latest balance information for all monitored process groups
func (ph *ProcessHunter) GetLatestPGroupsBalance() []ProcessGroupDayBalance {
	ph.pgroupsRWM.RLock()
	defer ph.pgroupsRWM.RUnlock()

	return ph.pgroups
}

// GetLatestProcessesBalance returns the latest time balance for all monitored processes
func (ph *ProcessHunter) GetLatestProcessesBalance() TimeBalance {
	ph.processesRWM.RLock()
	defer ph.processesRWM.RUnlock()

	return ph.processes
}

// GetBalance returns the complete balance history mapping dates to process time balances
func (ph *ProcessHunter) GetBalance() dayTimeBalance {
	ph.balanceRWM.RLock()
	defer ph.balanceRWM.RUnlock()

	return ph.balance
}

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
// It returns the activeSpec (an element of the specs array) and a boolean if such was found
// based on the current date and day of week
// it prioritizes more concrete, to more generic specifications, in the following order:
// - exact date, e.g. "2024-12-10"
// - a date from a list of days/dates, e.g. "2024-12-10 wed"
// - specific day of the week, e.g. "wed"
// - a day of the week, from a list of days, e.g. "mon wed fri"
// - any day "*"
func getActiveSpec(dt string, wd string, specs []string) (activeSpec string, found bool) {
	found = false

	dateInList := false
	dayMatch := false
	dayInList := false

	for _, k := range specs {
		if k == dt { // exact date match - end of search
			activeSpec = k
			found = true
			break
		}

		// Check if date is in a space-separated list
		if !dateInList {
			parts := strings.Fields(k)
			if slices.Contains(parts, dt) {
				activeSpec = k
				found = true
				dateInList = true
			}
		}

		if !dateInList {
			if k == wd { // day of week match
				activeSpec = k
				found = true
				dayMatch = true
			}
			if !dayMatch {
				// Check if day is in a space-separated list
				parts := strings.Fields(k)
				if slices.Contains(parts, wd) {
					activeSpec = k
					found = true
					dayInList = true
				}

				if !dayInList {
					if k == "*" {
						activeSpec = k
						found = true
					}
				}
			}
		}
	}
	return
}

// mapKeysToSlice extracts all keys from a map and returns them as a slice
func mapKeysToSlice[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// evalDayLimit returns the day time limit l and boolean defined that indicates if l is defined,
// based on the current date dt and week day wd, and the provided DayLimit spec dl
// See getActiveSpec to understand how a particular DayLimit is selected from dl based on dt and wd
func evalDayLimit(dt string, wd string, dl DayLimits) (l time.Duration, defined bool) {
	specs := mapKeysToSlice(dl)

	spec, found := getActiveSpec(dt, wd, specs)

	if found {
		defined = true
		l = dl[spec]
	}

	return
}

// isOvertime evaluates whether the balance exceeds the active day limit (if defined),
// based on the current date dt and week day wd, and the provided DayLimits spec dl.
// isOvertime returns overtime - the result of the evaluation, limit - the active day limit,
// and defined - that indicates if a limit is defined.
// See getActiveSpec to understand how a particular limit is selected from dl based on dt and wd.
func isOvertime(balance time.Duration, dt string, wd string, dl DayLimits) (overtime bool, limit time.Duration, defined bool) {
	limit, defined = evalDayLimit(dt, wd, dl)
	if defined {
		overtime = balance > limit
	}
	return
}

// isBlocked evaluates whether now is within a downtime period,
// based on the current date dt and week day wd, and the provided Downtime spec dnt.
// isBlocked returns blocked - the result of the evaluation and downtimeSpec - the active downtime specification
// See getActiveSpec to understand how a particular downtimeSpec is selected from dnt based on dt and wd.
func isBlocked(now time.Time, dt string, wd string, dnt Downtime) (blocked bool, downtimeSpec []string) {

	// Step 1: Extract all day/date specification keys from the Downtime map
	// These keys can be things like "*", "mon wed fri", "2024-12-25", etc.
	specs := mapKeysToSlice(dnt)

	// Step 2: Find the most specific matching spec for the current date and weekday
	// getActiveSpec prioritizes: exact date > date in list > specific day > day in list > wildcard "*"
	spec, found := getActiveSpec(dt, wd, specs)

	// Step 3: If a matching spec was found, check if current time falls within any downtime period
	if found {
		// Get the list of downtime periods for this spec (e.g., ["09:00..17:00", "..10:00"])
		downtimeSpec = dnt[spec]

		// Normalize the current time to just HH:MM format (strip date, seconds, etc.)
		// This allows direct time comparison without worrying about dates
		var err error
		now, err = time.Parse(dtTimeFormat, now.Format(dtTimeFormat))
		if err != nil {
			log.Printf("error normalizing time: %v", err)
			return
		}

		// Step 4: Check each downtime period to see if current time falls within it
		// Periods can be: "HH:MM..HH:MM" (range), "..HH:MM" (until), or "HH:MM.." (from)
		for _, period := range downtimeSpec {

			// Find the ".." separator that divides start and end times
			separator := strings.Index(period, "..")

			// Validate that the separator exists
			if separator < 0 {
				log.Printf("invalid downtime period format: %s (missing '..')", period)
				continue
			}

			// Assume we're in the period unless proven otherwise
			intersect := true

			// Check start time constraint (if present)
			// separator > 0 means there's a start time before ".."
			if separator > 0 {
				// Parse the start time (everything before "..")
				t, err := time.Parse(dtTimeFormat, period[0:separator])
				if err == nil {
					// If start time is after current time, we haven't reached the period yet
					if t.After(now) {
						intersect = false
					}
				} else {
					log.Printf("error parsing start time in downtime period %s: %v", period, err)
					continue
				}
			}

			// Check end time constraint (if present)
			// Validate bounds before slicing
			if separator+2 < len(period) {
				// Parse the end time (everything after "..")
				t, err := time.Parse(dtTimeFormat, period[separator+2:])
				if err == nil {
					// If end time is before current time, we've passed the period
					if t.Before(now) {
						intersect = false
					}
				} else {
					log.Printf("error parsing end time in downtime period %s: %v", period, err)
					continue
				}
			}

			// If current time falls within this period's constraints, we're blocked
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

// checkProcesses updates processes time balance (adding dt),
// checks for overtime and downtime and kills processes
func (ph *ProcessHunter) checkProcesses(ctx context.Context, dt time.Duration) error {

	// 0. reload config file, if necessary
	// ---------------
	reloaded, err := ph.reloadConfigIfNeeded()
	if err != nil {
		log.Println("error attempting to reload config:", err)
	}

	if reloaded {
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

	// Build a map of process names to PIDs for efficient lookup
	processPidMap := make(map[string][]int)
	for _, p := range pss {
		processName := p.Executable()
		ph.balance.add(date, processName, dt)
		processPidMap[processName] = append(processPidMap[processName], p.Pid())
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

	todayBalance := ph.balance[date]
	for groupIdx, groupLimit := range ph.limits { // iterate all processes day limits
		groupBalance := time.Duration(0)
		for _, processName := range groupLimit.PG { // iterate all processes in the process group
			groupBalance = groupBalance + todayBalance[processName]
			ph.processes[processName] = todayBalance[processName].Round(time.Second)
		}

		isOvertime, limit, defined := isOvertime(groupBalance, date, weekDay, groupLimit.DL)
		now = time.Now()
		isBlocked, activeDowntime := isBlocked(now, date, weekDay, groupLimit.DT)

		ph.pgroups[groupIdx] = ProcessGroupDayBalance{
			PG:           groupLimit.PG,
			Limit:        prettyDuration{limit},
			LimitDefined: defined,
			Balance:      prettyDuration{groupBalance.Round(time.Second)},
			Downtime:     activeDowntime,
			Blocked:      isBlocked,
			TimeStamp:    now.Format(dtTimeFormat),
		}

		// if overtime or blocked - kill the processes
		if isOvertime || isBlocked {
			log.Println(groupLimit.PG, ":", groupBalance, "/", limit)
			for _, processName := range groupLimit.PG { // iterate all processes in the process group
				if todayBalance[processName] > 0 {
					log.Println(processName, ":", todayBalance[processName])
					// Use the PID map for efficient lookup instead of iterating all processes
					if pids, exists := processPidMap[processName]; exists {
						for _, pid := range pids {
							// check if context is cancelled before attempting to kill
							select {
							case <-ctx.Done():
								return ctx.Err()
							default:
								log.Println("killing", pid)
								err := ph.killer(pid)
								if err != nil {
									log.Println("error killing", pid, ":", err.Error())
								}
							}
						}
					}
				}
			}
		} else {
			log.Println(groupLimit.PG, "remaining:", limit-groupBalance)
		}
	}

	// 3. Save time balance
	// ---------------
	ph.lastSavedRWM.RLock()
	shouldSave := ph.lastSaved.Add(ph.savePeriod).Before(time.Now())
	ph.lastSavedRWM.RUnlock()

	if shouldSave {
		if ph.balancePath != "" {
			log.Println("saving balance", ph.balancePath)
			err := ph.saveBalance()

			if err != nil {
				log.Println("error saving balance to", ph.balancePath, ":", err)
			} else {
				ph.lastSavedRWM.Lock()
				ph.lastSaved = time.Now()
				ph.lastSavedRWM.Unlock()
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

// add adds duration to the balance of the process processName for the day
func (dtb *dayTimeBalance) add(day string, processName string, duration time.Duration) {
	if _, dayExists := (*dtb)[day]; !dayExists {
		(*dtb)[day] = make(TimeBalance)
	}

	(*dtb)[day][processName] = (*dtb)[day][processName] + duration
}

// toText returns string representation of the date of t
func toText(t time.Time) string {
	return t.Format("2006-01-02")
}
