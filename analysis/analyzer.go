package analysis

import (
	"fmt"
	"time"

	"github.com/algao1/iv3/config"
	"github.com/algao1/iv3/fetcher"
	"github.com/algao1/iv3/store"
	"github.com/montanaflynn/stats"
	"go.uber.org/zap"
)

type PointsReader interface {
	ReadGlucosePoints(startTs, endTs int) ([]fetcher.GlucosePoint, error)
	ReadInsulinPoints(startTs, endTs int) ([]store.InsulinPoint, error)
	ReadCarbPoints(startTs, endTs int) ([]store.CarbPoint, error)
}

type Analyzer struct {
	reader PointsReader

	lowThreshold  int
	highThreshold int
	logger        *zap.Logger
}

func NewAnalyzer(reader PointsReader, cfg config.Iv3Config, logger *zap.Logger) *Analyzer {
	return &Analyzer{
		reader:        reader,
		lowThreshold:  cfg.LowThreshold,
		highThreshold: cfg.HighThreshold,
		logger:        logger,
	}
}

type DayToDayResult struct {
	Average float64
	InRange float64
	DtdAvg  []float64
}

func (a *Analyzer) DayToDay(startTs, endTs int) (*DayToDayResult, error) {
	glucosePoints, err := a.reader.ReadGlucosePoints(startTs, endTs)
	if err != nil {
		return nil, fmt.Errorf("failed to read glucose points: %w", err)
	}

	inRange := 0.0
	buckets := make([][]fetcher.GlucosePoint, 24*12)
	glucoseValues := make([]float64, len(glucosePoints))

	for i, point := range glucosePoints {
		truncated := point.Time.Truncate(5 * time.Minute)
		bucket := truncated.Hour()*12 + truncated.Minute()/5
		buckets[bucket] = append(buckets[bucket], point)

		if int(point.Value) >= a.lowThreshold &&
			int(point.Value) <= a.highThreshold {
			inRange += 1
		}

		glucoseValues[i] = point.Value
	}

	avg, _ := stats.Mean(glucoseValues)

	bucketAvg := make([]float64, len(buckets))
	for i, bucket := range buckets {
		if len(bucket) == 0 {
			continue
		}
		for _, point := range bucket {
			bucketAvg[i] += point.Value
		}
		bucketAvg[i] /= float64(len(bucket))
	}

	return &DayToDayResult{
		Average: avg,
		InRange: inRange / float64(len(glucosePoints)),
		DtdAvg:  bucketAvg,
	}, nil
}
