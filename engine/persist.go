package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

// MarshalJSON marshals pgdb using 12h35m46s duration format
func (pd prettyDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(pd.String())
}

// MarshalJSON marshals tb using 12h35m46s duration format
func (tb TimeBalance) MarshalJSON() ([]byte, error) {
	aux := make(map[string]string)

	for k, v := range tb {
		aux[k] = time.Duration.String(v)
	}

	return json.Marshal(aux)
}

// UnmarshalJSON unmarshals dtl using 12h35m46s duration format
func (tb *TimeBalance) UnmarshalJSON(data []byte) error {
	aux := make(map[string]string)

	err := json.Unmarshal(data, &aux)

	if err != nil {
		return err
	}

	(*tb) = make(TimeBalance)

	for k, v := range aux {
		l, err := time.ParseDuration(v)
		if err != nil {
			break
		}
		(*tb)[k] = l
	}

	return err
}

// MarshalJSON marshals dtl using 12h35m46s duration format
func (dtl DailyLimits) MarshalJSON() ([]byte, error) {
	aux := make(map[string]string)

	for k, v := range dtl {
		aux[k] = time.Duration.String(v)
	}

	return json.Marshal(aux)
}

// UnmarshalJSON unmarshals dtl, lowercasing the key and using 12h35m46s duration format
func (dtl *DailyLimits) UnmarshalJSON(data []byte) error {
	aux := make(map[string]string)

	err := json.Unmarshal(data, &aux)

	if err != nil {
		return err
	}

	(*dtl) = make(DailyLimits)

	for k, v := range aux {
		l, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		(*dtl)[strings.ToLower(k)] = l // converts to lower caps
	}

	return nil
}

// isValidDailyLimitsFormat checks whether string with daily limits is correct
func isValidDailyLimitsFormat(l DailyLimits) bool {
	for k := range l {
		if k == "*" {
			continue
		}

		words := strings.Fields(k)

		if len(words) == 0 {
			return false
		}

		for _, w := range words {
			valid := false

			// week days
			for _, d := range weekDays {
				if w == d {
					valid = true
					break
				}
			}

			// dates
			match, err := regexp.MatchString(`^\d{1,4}-\d{1,2}-\d{1,2}$`, w)
			if match && err == nil {
				valid = true
				break
			}

			if !valid {
				return false
			}
		}
	}
	return true
}

// parseConfig parses configuration from b, represented as JSON
func parseConfig(b []byte) ([]ProcessGroupDailyLimit, error) {
	var limits []ProcessGroupDailyLimit

	err := json.Unmarshal(b, &limits)
	if err != nil {
		return nil, err
	}

	for _, l := range limits {
		if len(l.PG) == 0 {
			return nil, errors.New(fmt.Sprintln("Process list required"))
		}
		if len(l.DL) == 0 {
			return nil, errors.New(fmt.Sprintln("Daily limits required"))
		}
		if !isValidDailyLimitsFormat(l.DL) {
			return nil, errors.New(fmt.Sprintln("Bad date or days of the week format:", l.DL))
		}
	}

	return limits, nil
}

// crc64Table is used in crc64.Checksum
var crc32Table = crc32.MakeTable(crc32.Koopman)

// setLimits sets ph.limits, ph.cfgTime and ph.limitsHash
func (ph *ProcessHunter) setLimits(limits []ProcessGroupDailyLimit) error {
	ph.limits = limits

	b, err := json.Marshal(ph.limits)
	if err != nil {
		log.Panicln("cannot marshal limits to json")
	}
	ph.limitsHash = crc32.Checksum(b, crc32Table)

	file, err := os.Stat(ph.cfgPath)
	if err != nil {
		return err
	}

	ph.cfgTime = file.ModTime()
	return nil
}

// SetConfig sets configuration b (represented as json) and saves it to the ph.cfgPath.
// if ph.cfgPath cannot be written, the call fails and new config is not set
func (ph *ProcessHunter) SetConfig(b []byte) error {
	limits, err := parseConfig(b)
	if err != nil {
		return err
	}

	ph.limitsRWM.Lock()
	defer ph.limitsRWM.Unlock()

	err = ioutil.WriteFile(ph.cfgPath, b, 0644)
	if err != nil {
		return err
	}

	return ph.setLimits(limits)
}

// LoadConfig loads ProcessHunder configuration from path
func (ph *ProcessHunter) LoadConfig() error {
	b, err := ioutil.ReadFile(ph.cfgPath)
	if err != nil {
		return err
	}

	limits, err := parseConfig(b)
	if err != nil {
		return err
	}

	ph.limitsRWM.Lock()
	defer ph.limitsRWM.Unlock()

	return ph.setLimits(limits)
}

// LoadBalance loads the balance from provided path
func (ph *ProcessHunter) LoadBalance() error {
	ph.balanceRWM.Lock()
	defer ph.balanceRWM.Unlock()

	ph.balance = make(dailyTimeBalance)

	b, err := ioutil.ReadFile(ph.balancePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, &ph.balance)
}

// saveBalance saves balance to ph.balancePath
func (ph *ProcessHunter) saveBalance() error {
	d, err := json.MarshalIndent(ph.balance, "", "\t")

	if err != nil {
		return err
	}

	return ioutil.WriteFile(ph.balancePath, d, 0644)
}

// SaveBalance saves balance in a thread-safe way
func (ph *ProcessHunter) SaveBalance() error {
	ph.balanceRWM.RLock()
	defer ph.balanceRWM.RUnlock()

	return ph.saveBalance()
}
