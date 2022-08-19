package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type TimeSeriesJson struct {
	Base  string                        `json:"base"`
	Rates map[string]map[string]float64 `json:"rates"`
}

func FetchRange(base string, from, to time.Time, symbols ...string) (*data.Frame, error) {
	url := fmt.Sprintf("https://api.exchangerate.host/timeseries?start_date=%s&end_date=%s&base=%s&symbols=%s",
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
		base,
		strings.Join(symbols, ","),
	)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var rates TimeSeriesJson
	err = json.NewDecoder(resp.Body).Decode(&rates)
	if err != nil {
		return nil, err
	}

	log.DefaultLogger.Info("FetchRange", "timeseries", rates)

	out := data.NewFrame("response")

	times := make([]time.Time, 0, len(rates.Rates))

	// we first filter out all the times and sort them
	for rawWhen := range rates.Rates {
		when, err := time.Parse("2006-01-02", rawWhen)
		if err != nil {
			continue
		}

		if when.After(from) && when.Before(to) {
			times = append(times, when)
		}
	}

	sort.SliceStable(times, func(i, j int) bool { return times[i].Unix() < times[j].Unix() })

	out.Fields = append(out.Fields, data.NewField("time", nil, times))

	for _, symbol := range symbols {
		exchangeRate := make([]float64, 0, len(times))

		for _, when := range times {
			exchangeRate = append(exchangeRate, rates.Rates[when.Format("2006-01-02")][symbol])
		}

		out.Fields = append(out.Fields, data.NewField(symbol, nil, exchangeRate))
	}

	return out, nil
}
