package alert

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/algao1/iv3/config"
	"github.com/algao1/iv3/store"
	"go.uber.org/zap"
)

// TODO: Have a better way for dealing with unit conversions...
// Maybe only do it for outputs?

const (
	PredLowGlucoseEvent     = "pred_low_glucose"
	HighGlucoseEvent        = "high_glucose"
	MissingLongInsulinEvent = "missing_long_insulin"

	PredLowGlucoseWindow     = 5 * time.Minute
	HighGlucoseWindow        = 45 * time.Minute
	MissingLongInsulinWindow = 1 * time.Hour
)

type AlertingReadWriter interface {
	ReadGlucosePoints(startTs, endTs int) ([]store.GlucosePoint, error)
	ReadInsulinPoints(startTs, endTs int) ([]store.InsulinPoint, error)
	ReadCarbPoints(startTs, endTs int) ([]store.CarbPoint, error)
	WriteEventPoint(point store.EventPoint) error
	ReadEventPoints(startTs, endTs int) ([]store.EventPoint, error)
}

type Alerter struct {
	rw AlertingReadWriter

	// Configs.
	unit                 string
	insPeriodType        map[string]string
	endpoint             string
	missingLongThreshold time.Duration
	lowThreshold         int
	highThreshold        int

	logger *zap.Logger
}

func NewAlerter(rw AlertingReadWriter, cfg config.Iv3Config,
	insCfg []config.InsulinConfig, logger *zap.Logger) *Alerter {
	a := &Alerter{
		rw:                   rw,
		unit:                 cfg.Unit,
		insPeriodType:        make(map[string]string),
		endpoint:             cfg.Endpoint,
		missingLongThreshold: time.Duration(cfg.MissingLongThreshold) * time.Hour,
		lowThreshold:         cfg.LowThreshold,
		highThreshold:        cfg.HighThreshold,
		logger:               logger,
	}
	for _, ins := range insCfg {
		a.insPeriodType[ins.Name] = ins.PeriodType
	}

	logger.Info("started Alerter",
		zap.Duration("missingLongThreshold", a.missingLongThreshold),
		zap.Int("lowThreshold", a.lowThreshold),
	)

	go a.run()
	return a
}

func (a *Alerter) run() {
	ticker := time.NewTicker(30 * time.Second)
	for range ticker.C {
		a.checkPredictedGlucose()
		a.checkMissingLongInsulin()
		a.checkHighGlucose()
	}
}

func (a *Alerter) checkPredictedGlucose() {
	windowStart := time.Now().Add(-30 * time.Minute)
	windowEnd := time.Now()

	points, err := a.rw.ReadGlucosePoints(
		int(windowStart.Unix()),
		int(windowEnd.Unix()),
	)
	if err != nil {
		a.logger.Error("error reading glucose points", zap.Error(err))
		return
	}
	if len(points) < 3 {
		a.logger.Info("not enough points to predict glucose")
		return
	}

	// Predict 20 minutes out using a simple trend.
	lastPoints := points[len(points)-3:]
	trend := (lastPoints[2].Value - lastPoints[0].Value) / 2
	predValue := lastPoints[2].Value + trend*4
	a.logger.Debug("predicted glucose", zap.Float64("value", predValue))

	if predValue < float64(a.lowThreshold) &&
		a.noEventsInPast(PredLowGlucoseEvent, PredLowGlucoseWindow) {
		alert := Alert{
			Title:    "Incoming Low Glucose",
			Event:    PredLowGlucoseEvent,
			Message:  fmt.Sprintf("Glucose is predicted to be %.2f in 20 minutes", a.molarOrMass(predValue)),
			Priority: "high",
		}
		if err = a.publishAlert(alert); err != nil {
			a.logger.Error("unable to publish alert", zap.Error(err))
		}
	}
}

func (a *Alerter) checkMissingLongInsulin() {
	windowStart := time.Now().Add(-a.missingLongThreshold)
	windowEnd := time.Now()

	points, err := a.rw.ReadInsulinPoints(
		int(windowStart.Unix()),
		int(windowEnd.Unix()),
	)
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
		Event:    MissingLongInsulinEvent,
		Message:  fmt.Sprintf("No long insulin in the past %s hours", a.missingLongThreshold),
		Priority: "high",
	}
	if err = a.publishAlert(alert); err != nil {
		a.logger.Error("unable to publish alert", zap.Error(err))
	}
}

func (a *Alerter) checkHighGlucose() {
	windowStart := time.Now().Add(-10 * time.Minute)
	windowEnd := time.Now()

	points, err := a.rw.ReadGlucosePoints(
		int(windowStart.Unix()),
		int(windowEnd.Unix()),
	)
	if err != nil {
		a.logger.Error("error reading glucose points", zap.Error(err))
		return
	}
	if len(points) == 0 {
		a.logger.Error("no glucose points found")
		return
	}

	curGlucose := points[len(points)-1].Value
	if curGlucose <= float64(a.highThreshold) {
		return
	}
	if !a.noEventsInPast(HighGlucoseEvent, HighGlucoseWindow) {
		return
	}

	alert := Alert{
		Title: "High Glucose",
		Event: HighGlucoseEvent,
		Message: fmt.Sprintf("Glucose is %.1f and above target %.1f",
			a.molarOrMass(curGlucose),
			a.molarOrMass(float64(a.highThreshold)),
		),
		Priority: "high",
	}
	if err = a.publishAlert(alert); err != nil {
		a.logger.Error("unable to publish alert", zap.Error(err))
	}
}

func (a *Alerter) noEventsInPast(event string, d time.Duration) bool {
	windowStart, windowEnd := time.Now().Add(-d), time.Now()
	points, err := a.rw.ReadEventPoints(
		int(windowStart.Unix()),
		int(windowEnd.Unix()),
	)
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
	Event    string
	Message  string
	Priority string
	Tags     []string
}

func (a *Alerter) publishAlert(alert Alert) error {
	req, err := http.NewRequest("POST", "https://ntfy.sh/"+a.endpoint, strings.NewReader(alert.Message))
	if err != nil {
		return fmt.Errorf("unable to make request: %w", err)
	}

	if alert.Title != "" {
		req.Header.Set("Title", alert.Title)
	}
	if alert.Priority != "" {
		req.Header.Set("Priority", alert.Priority)
	}
	if len(alert.Tags) > 0 {
		req.Header.Set("Tags", strings.Join(alert.Tags, ","))
	}

	_, err = http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to send request: %w", err)
	}

	a.logger.Info(
		"published alert",
		zap.String("title", alert.Title),
		zap.String("message", alert.Message),
	)

	err = a.rw.WriteEventPoint(store.EventPoint{
		Event:   alert.Event,
		Message: alert.Message,
		Time:    time.Now(),
	})
	if err != nil {
		return fmt.Errorf("unable to write event to database: %w", err)
	}
	return nil
}

func (a *Alerter) molarOrMass(value float64) float64 {
	if a.unit == "mmol/L" {
		return value / 18
	}
	return value
}
