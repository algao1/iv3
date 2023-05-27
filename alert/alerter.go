package alert

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/algao1/iv3/config"
	"github.com/algao1/iv3/fetcher"
	"github.com/algao1/iv3/store"
	"go.uber.org/zap"
)

const (
	PredLowGlucoseEvent  = "pred_low_glucose"
	PredLowGlucoseWindow = 5 * time.Minute

	MissingLongInsulinEvent  = "missing_long_insulin"
	MissingLongInsulinWindow = 1 * time.Hour
)

type AlertingReadWriter interface {
	ReadGlucosePoints(startTs, endTs int) ([]fetcher.GlucosePoint, error)
	ReadInsulinPoints(startTs, endTs int) ([]store.InsulinPoint, error)
	ReadCarbPoints(startTs, endTs int) ([]store.CarbPoint, error)

	WriteEventPoint(point store.EventPoint) error
	ReadEventPoints(startTs, endTs int) ([]store.EventPoint, error)
}

type Alerter struct {
	rw AlertingReadWriter

	// Configs.
	insPeriodType map[string]string
	endpoint      string

	missingLongThreshold time.Duration
	lowThreshold         int

	logger *zap.Logger
}

func NewAlerter(rw AlertingReadWriter, cfg config.AlertConfig,
	insCfg []config.InsulinConfig, logger *zap.Logger) *Alerter {
	a := &Alerter{
		rw:                   rw,
		insPeriodType:        make(map[string]string),
		endpoint:             cfg.Endpoint,
		missingLongThreshold: time.Duration(cfg.MissingLongThreshold) * time.Hour,
		lowThreshold:         cfg.LowThreshold,
		logger:               logger,
	}
	for _, ins := range insCfg {
		a.insPeriodType[ins.Name] = ins.PeriodType
	}

	logger.Info("started Alerter",
		zap.Duration("missingLongThreshold", a.missingLongThreshold),
		zap.Int("lowThreshold", a.lowThreshold),
	)

	go a.check()
	return a
}

func (a *Alerter) check() {
	predTicker := time.NewTicker(30 * time.Second)
	missingLongTicker := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-predTicker.C:
			a.checkPredGlucose()
		case <-missingLongTicker.C:
			a.checkMissingLongInsulin()
		}
	}
}

func (a *Alerter) checkPredGlucose() {
	windowStart, windowEnd := time.Now().Add(-30*time.Minute), time.Now()

	points, err := a.rw.ReadGlucosePoints(int(windowStart.Unix()), int(windowEnd.Unix()))
	if err != nil {
		a.logger.Error("error reading glucose points", zap.Error(err))
		return
	}
	if len(points) < 3 {
		a.logger.Info("not enough glucose points to check")
		return
	}

	// Predict 20 minutes out using a simple trend.
	lastPoints := points[len(points)-3:]
	trend := (lastPoints[2].Value - lastPoints[0].Value) / 2
	predValue := lastPoints[2].Value + trend*4
	a.logger.Debug("predicted glucose", zap.Float64("value", predValue))

	if predValue < float64(a.lowThreshold) && a.noEventsInPast(PredLowGlucoseEvent, PredLowGlucoseWindow) {
		alert := Alert{
			Title:    "Incoming Low Glucose",
			Message:  fmt.Sprintf("Glucose is predicted to be %.2f in 20 minutes", predValue/18),
			Priority: "high",
		}
		a.publishAlert(alert)
		a.rw.WriteEventPoint(store.EventPoint{
			Event:   PredLowGlucoseEvent,
			Message: alert.Message,
			Time:    time.Now(),
		})
	}
}

func (a *Alerter) checkMissingLongInsulin() {
	windowStart := time.Now().Add(-a.missingLongThreshold)
	windowEnd := time.Now()

	points, err := a.rw.ReadInsulinPoints(int(windowStart.Unix()), int(windowEnd.Unix()))
	if err != nil {
		a.logger.Error("error reading insulin points", zap.Error(err))
		return
	}

	for _, point := range points {
		if a.insPeriodType[point.Type] == "long" {
			return
		}
	}

	if !a.noEventsInPast(MissingLongInsulinEvent, MissingLongInsulinWindow) {
		return
	}

	alert := Alert{
		Title:    "Missing Long Insulin",
		Message:  fmt.Sprintf("No long insulin in the past %s hours", a.missingLongThreshold),
		Priority: "high",
	}
	a.publishAlert(alert)
	a.rw.WriteEventPoint(store.EventPoint{
		Event:   MissingLongInsulinEvent,
		Message: alert.Message,
		Time:    time.Now(),
	})
}

func (a *Alerter) noEventsInPast(event string, d time.Duration) bool {
	windowStart, windowEnd := time.Now().Add(-d), time.Now()
	points, err := a.rw.ReadEventPoints(int(windowStart.Unix()), int(windowEnd.Unix()))
	if err != nil {
		a.logger.Error("error reading event points", zap.Error(err))
		return false
	}

	for _, point := range points {
		if point.Event == event {
			return false
		}
	}
	return true
}

type Alert struct {
	Title    string
	Message  string
	Priority string
	Tags     []string
}

func (a *Alerter) publishAlert(alert Alert) {
	req, _ := http.NewRequest("POST", "https://ntfy.sh/"+a.endpoint, strings.NewReader(alert.Message))
	if alert.Title != "" {
		req.Header.Set("Title", alert.Title)
	}
	if alert.Priority != "" {
		req.Header.Set("Priority", alert.Priority)
	}
	if len(alert.Tags) > 0 {
		req.Header.Set("Tags", strings.Join(alert.Tags, ","))
	}

	http.DefaultClient.Do(req)
	a.logger.Info(
		"published alert",
		zap.String("title", alert.Title),
		zap.String("message", alert.Message),
	)
}
