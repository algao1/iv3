package store

import (
	"context"
	"fmt"

	"github.com/algao1/iv3/fetcher"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/influxdata/influxdb-client-go/v2/domain"
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

func NewInfluxDB(token, url string, logger *zap.Logger) (*InfluxDBClient, error) {
	client := &InfluxDBClient{
		client: influxdb2.NewClient(url, token),
		logger: logger,
		token:  token,
		url:    url,
	}

	// Temporary, will set timeout someday...
	ctx := context.Background()

	_, err := client.client.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to ping InfluxDB server: %w", err)
	}
	logger.Info("started InfluxDB client", zap.String("token", token), zap.String("url", url))

	bucketsAPI := client.client.BucketsAPI()
	orgs, err := client.client.OrganizationsAPI().GetOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get organizations: %w", err)
	}

	var iv3Org domain.Organization
	for _, org := range *orgs {
		if org.Name == Org {
			iv3Org = org
			break
		}
	}

	var bucketNames = []string{GlucoseBucket, InsulinBucket, EventsBucket}
	for _, bucketName := range bucketNames {
		_, err := bucketsAPI.FindBucketByName(context.Background(), bucketName)
		if err == nil {
			continue
		}
		_, err = bucketsAPI.CreateBucketWithName(context.Background(), &iv3Org, bucketName)
		if err != nil {
			return nil, fmt.Errorf("unable to create bucket %s: %w", bucketName, err)
		}
	}

	return client, nil
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
            |> range(start: %d, stop: %d)
            |> filter(fn: (r) => r["_field"] == "value")
            |> yield()
    `, GlucoseBucket, startTs, endTs)

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

func (c *InfluxDBClient) WriteInsulinPoint(insulin fetcher.InsulinPoint) error {
	writeAPI := c.client.WriteAPIBlocking(Org, InsulinBucket)
	fields := map[string]any{
		"value": insulin.Value,
		"type":  insulin.Type,
	}
	tags := map[string]string{
		"type": insulin.Type,
	}
	point := write.NewPoint("insulin", tags, fields, insulin.Time)

	err := writeAPI.WritePoint(context.Background(), point)
	if err != nil {
		return fmt.Errorf("unable to write insulin point to InfluxDB: %w", err)
	}
	c.logger.Debug("wrote insulin point", zap.Time("ts", point.Time()), zap.Any("fields", fields))
	return nil
}

func (c *InfluxDBClient) ReadInsulinPoints(startTs, endTs int) ([]fetcher.InsulinPoint, error) {
	queryAPI := c.client.QueryAPI(Org)
	fluxQuery := fmt.Sprintf(`
        data = from(bucket: "%s")
            |> range(start: %d, stop: %d)
            |> filter(fn: (r) => r["_field"] == "value")
            |> yield()
    `, InsulinBucket, startTs, endTs)

	result, err := queryAPI.Query(context.Background(), fluxQuery)
	if err != nil {
		return nil, fmt.Errorf("unable to read insulin points between %d and %d: %w", startTs, endTs, err)
	}

	insulin := make([]fetcher.InsulinPoint, 0)
	for result.Next() {
		insulin = append(insulin, fetcher.InsulinPoint{
			Value: int(result.Record().Value().(int64)),
			Type:  result.Record().ValueByKey("type").(string),
			Time:  result.Record().Time(),
		})
	}
	return insulin, nil
}
