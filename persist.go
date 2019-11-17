package ph

import (
	"encoding/json"
	"io/ioutil"
)

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
