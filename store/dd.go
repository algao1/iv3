package store

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV2"
	"github.com/algao1/iv3/config"
)

type DDClient struct {
	apiClient *datadog.APIClient
	iv3cfg    *config.Iv3Config
}

func NewDDClient(cfg *config.Iv3Config) *DDClient {
	configuration := datadog.NewConfiguration()
	apiClient := datadog.NewAPIClient(configuration)
	return &DDClient{
		apiClient: apiClient,
		iv3cfg:    cfg,
	}
}

func (dd *DDClient) WriteGlucosePoints(glucose []GlucosePoint) error {
	body := datadogV2.MetricPayload{
		Series: []datadogV2.MetricSeries{
			{
				Metric: "iv3.cur_glucose",
				Type:   datadogV2.METRICINTAKETYPE_UNSPECIFIED.Ptr(),
				Points: []datadogV2.MetricPoint{},
				Tags:   []string{"status:above_range"},
			},
			{
				Metric: "iv3.cur_glucose",
				Type:   datadogV2.METRICINTAKETYPE_UNSPECIFIED.Ptr(),
				Points: []datadogV2.MetricPoint{},
				Tags:   []string{"status:below_range"},
			},
			{
				Metric: "iv3.cur_glucose",
				Type:   datadogV2.METRICINTAKETYPE_UNSPECIFIED.Ptr(),
				Points: []datadogV2.MetricPoint{},
				Tags:   []string{"status:in_range"},
			},
		},
	}

	for _, point := range glucose {
		if point.Value > float64(dd.iv3cfg.HighThreshold) {
			body.Series[0].Points = append(body.Series[0].Points, datadogV2.MetricPoint{
				Value:     datadog.PtrFloat64(point.Value),
				Timestamp: datadog.PtrInt64(point.Time.Unix()),
			})
		} else if point.Value < float64(dd.iv3cfg.LowThreshold) {
			body.Series[1].Points = append(body.Series[1].Points, datadogV2.MetricPoint{
				Value:     datadog.PtrFloat64(point.Value),
				Timestamp: datadog.PtrInt64(point.Time.Unix()),
			})
		} else {
			body.Series[2].Points = append(body.Series[2].Points, datadogV2.MetricPoint{
				Value:     datadog.PtrFloat64(point.Value),
				Timestamp: datadog.PtrInt64(point.Time.Unix()),
			})
		}
	}

	ctx := datadog.NewDefaultContext(context.Background())
	api := datadogV2.NewMetricsApi(dd.apiClient)
	_, _, err := api.SubmitMetrics(ctx, body, *datadogV2.NewSubmitMetricsOptionalParameters())
	if err != nil {
		return fmt.Errorf("unable to call `MetricsApi.SubmitMetrics`: %w", err)
	}
	return nil
}
