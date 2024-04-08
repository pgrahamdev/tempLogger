package types

import (
	"fmt"
	"time"
)

type THData struct {
	ID         string  `json:"id"`
	TimeStamp  string  `json:"timestamp"`
	Humidity   float32 `json:"humidity"`
	TempC      float32 `json:"tempC"`
	TempF      float32 `json:"tempF"`
	HeatIndexC float32 `json:"heatIndexC"`
	HeatIndexF float32 `json:"heatIndexF"`
}

type TLCfg struct {
	ID         string
	SerialPath string
	Baud       int
	Retries    int
}

type TempRecord struct {
	Date  int64   `json:"date"`
	Value float32 `json:"value"`
}

func (tr *TempRecord) FromTHData(thd THData) error {
	tmpTime, err := time.Parse(time.RFC3339, thd.TimeStamp)
	if err != nil {
		err = fmt.Errorf("FromTHData: Timestamp parse error: %s: %w",
			thd.TimeStamp, err)
		return err
	}
	tr.Date = tmpTime.UnixMilli()
	tr.Value = thd.TempF
	return nil
}

type TLSummary struct {
	Name     string       `json:"name"`
	MaxTemp  TempRecord   `json:"max"`
	MinTemp  TempRecord   `json:"min"`
	LastTemp TempRecord   `json:"last"`
	TLData   []TempRecord `json:"tlData"`
}

type TLPage struct {
	Page      string      `json:"page"`
	Summaries []TLSummary `json:"summaries"`
}
