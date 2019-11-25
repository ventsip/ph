package lib

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

// UnmarshalJSON unmarshales dtl using 12h35m46s duration format
func (dtl *DailyTimeLimit) UnmarshalJSON(data []byte) error {
	type Alias DailyTimeLimit

	aux := &struct {
		L string `json:"limit"`
		*Alias
	}{
		Alias: (*Alias)(dtl),
	}

	err := json.Unmarshal(data, &aux)

	if err != nil {
		return err
	}

	dtl.L, err = time.ParseDuration(aux.L)

	return err
}

// LoadConfig loads ProcessHunder configuration from path
func (ph *ProcessHunter) LoadConfig(path string) error {
	ph.limits = nil

	b, err := ioutil.ReadFile(path)

	if err != nil {
		return err
	}

	return json.Unmarshal(b, &ph.limits)
}

// LoadBalance loads the balance from provided path
func (ph *ProcessHunter) LoadBalance(path string) error {
	ph.balance = make(dailyTimeBalance)

	b, err := ioutil.ReadFile(path)

	if err != nil {
		return err
	}

	return json.Unmarshal(b, &ph.balance)
}

// SaveBalance saves balance to provided path
func (ph *ProcessHunter) SaveBalance(path string) error {
	d, err := json.MarshalIndent(ph.balance, "", "\t")

	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, d, 0644)
}
