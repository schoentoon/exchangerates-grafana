package plugin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// Make sure SampleDatasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler, backend.StreamHandler interfaces. Plugin should not
// implement all these interfaces - only those which are required for a particular task.
// For example if plugin does not need streaming functionality then you are free to remove
// methods that implement backend.StreamHandler. Implementing instancemgmt.InstanceDisposer
// is useful to clean up resources used by previous datasource instance when a new datasource
// instance created upon datasource settings changed.
var (
	_ backend.QueryDataHandler      = (*ExchangeRatesDataSource)(nil)
	_ instancemgmt.InstanceDisposer = (*ExchangeRatesDataSource)(nil)
)

// NewExchangeRatesDatasource creates a new datasource instance.
func NewExchangeRatesDatasource(_ backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return &ExchangeRatesDataSource{
		httpClient: http.DefaultClient,
	}, nil
}

// ExchangeRatesDataSource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type ExchangeRatesDataSource struct {
	httpClient *http.Client
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *ExchangeRatesDataSource) Dispose() {
	// Clean up datasource instance resources.
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *ExchangeRatesDataSource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	log.DefaultLogger.Info("QueryData called", "request", req)

	// create response struct
	response := backend.NewQueryDataResponse()

	type task struct {
		d backend.DataResponse
		q backend.DataQuery
	}

	ch := make(chan task, len(req.Queries))
	for _, q := range req.Queries {
		go func(q backend.DataQuery) {
			ch <- task{d: d.query(ctx, req.PluginContext, q), q: q}
		}(q)
	}

	for range req.Queries {
		select {
		case task := <-ch:
			// save the response in a hashmap
			// based on with RefID as identifier
			response.Responses[task.q.RefID] = task.d
		case <-ctx.Done():
			// if the context finishes before all the requests we just bail out
			return response, nil
		}
	}

	return response, nil
}

type queryModel struct {
	BaseCurrency string `json:"baseCurrency"`
	ToCurrency   string `json:"toCurrency"`
}

func (d *ExchangeRatesDataSource) query(_ context.Context, pCtx backend.PluginContext, query backend.DataQuery) backend.DataResponse {
	log.DefaultLogger.Info("query", "json", query.JSON)
	response := backend.DataResponse{}

	// Unmarshal the JSON into our queryModel.
	var qm queryModel

	response.Error = json.Unmarshal(query.JSON, &qm)
	if response.Error != nil {
		return response
	}

	frame, err := d.fetchRange(qm.BaseCurrency, query.TimeRange.From, query.TimeRange.To, qm.ToCurrency)
	if err != nil {
		response.Error = err
		return response
	}

	// add the frames to the response.
	response.Frames = append(response.Frames, frame)

	return response
}
