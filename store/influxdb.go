package store

import (
	"context"
	"fmt"

	"github.com/algao1/iv3/fetcher"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"go.uber.org/zap"
)

const (
	Org           = "iv3"
	GlucoseBucket = "iv3_glucose"
	InsulinBucket = "iv3_insulin"
	EventsBucket  = "iv3_events"
)

type InfluxDBClient struct {
	client influxdb2.Client
	logger *zap.Logger

	token string
	url   string
}

func NewInfluxDB(token, url string, logger *zap.Logger) *InfluxDBClient {
	client := &InfluxDBClient{
		client: influxdb2.NewClient(url, token),
		logger: logger,
		token:  token,
		url:    url,
	}

	_, err := client.client.Ping(context.Background())
	if err != nil {
		logger.Info("unable to ping InfluxDB server", zap.Error(err))
	}
	logger.Info("started InfluxDB client", zap.String("token", token), zap.String("url", url))

	return client
}

func (c *InfluxDBClient) WriteGlucosePoints(glucose []fetcher.GlucosePoint) error {
	writeAPI := c.client.WriteAPIBlocking(Org, GlucoseBucket)
	for _, gp := range glucose {
		fields := map[string]any{
			"value": gp.Value,
			"trend": gp.Trend,
		}
		point := write.NewPoint("glucose", map[string]string{}, fields, gp.Time)

		err := writeAPI.WritePoint(context.Background(), point)
		if err != nil {
			return fmt.Errorf("unable to write glucose point to InfluxDB: %w", err)
		}
		c.logger.Debug("wrote glucose point", zap.Time("ts", point.Time()), zap.Any("fields", fields))
	}
	return nil
}

func (c *InfluxDBClient) ReadGlucosePoints(startTs, endTs int) ([]fetcher.GlucosePoint, error) {
	queryAPI := c.client.QueryAPI(Org)
	// Currently, this only grabs all the values, and not the trends.
	// I am thinking if that is needed, we will need to remove the filter.
	fluxQuery := fmt.Sprintf(`
        data = from(bucket: "%s")
            |> range(start: %d)
            |> filter(fn: (r) => r["_field"] == "value")
            |> yield()
    `, GlucoseBucket, startTs)

	result, err := queryAPI.Query(context.Background(), fluxQuery)
	if err != nil {
		return nil, fmt.Errorf("unable to read glucose points between %d and %d: %w", startTs, endTs, err)
	}

	glucose := make([]fetcher.GlucosePoint, 0)
	for result.Next() {
		glucose = append(glucose, fetcher.GlucosePoint{
			Value: result.Record().Value().(float64),
			Time:  result.Record().Time(),
		})
	}
	return glucose, nil
}
