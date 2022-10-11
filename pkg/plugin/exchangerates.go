package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

type TimeSeriesJson struct {
	Base  string                        `json:"base"`
	Rates map[string]map[string]float64 `json:"rates"`
}

type Rates struct {
	Rates map[time.Time]float64
	Order []time.Time
}

func (r *Rates) Size() int64 {
	// 24 is the size of a time.Time, 8 is the size of float64.. we just multiply it with the amount of items
	// then we do the same trick for the Order array
	return int64(((24 + 8) * len(r.Rates)) + (24 * len(r.Order)))
}

func (d *ExchangeRatesDataSource) fetchRange(base string, from, to time.Time, symbol string) (*Rates, error) {
	var out *Rates
	var err error
	key := fmt.Sprintf("%s-%s", base, symbol)
	val, found := d.cache.Get(key)
	if found {
		// TODO: add checks here on whether the timerange requested is actually in the cache, if not we request it and extend the cache
		log.DefaultLogger.Info("We got our data from cache..")
		return val.(*Rates), nil
	}

	url := fmt.Sprintf("https://api.exchangerate.host/timeseries?start_date=%s&end_date=%s&base=%s&symbols=%s",
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
		base,
		symbol,
	)
	resp, err := d.httpClient.Get(url)
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

	out = &Rates{
		Rates: make(map[time.Time]float64, len(rates.Rates)),
		Order: make([]time.Time, 0, len(rates.Rates)),
	}

	for whenRaw, rate := range rates.Rates {
		when, err := time.Parse("2006-01-02", whenRaw)
		if err != nil {
			continue
		}

		out.Rates[when] = rate[symbol]
		out.Order = append(out.Order, when)
	}

	sort.SliceStable(out.Order, func(i, j int) bool { return out.Order[i].Unix() < out.Order[j].Unix() })

	d.cache.SetWithTTL(key, out, out.Size(), calcTTL())

	return out, nil
}
