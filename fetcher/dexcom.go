package fetcher

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	appID           = "d89443d2-327c-4a6f-89e5-496bbb0317db"
	baseUrl         = "https://shareous1.dexcom.com/ShareWebServices/Services"
	loginEndpoint   = "General/LoginPublisherAccountByName"
	authEndpoint    = "General/AuthenticatePublisherAccount"
	glucoseEndpoint = "Publisher/ReadPublisherLatestGlucoseValues"

	minuteMax = "1440"
	countMax  = "288"
)

type GlucosePointsWriter interface {
	WriteGlucosePoints(glucose []GlucosePoint) error
}

type DexcomClient struct {
	client  *http.Client
	writers []GlucosePointsWriter
	logger  *zap.Logger

	accountName string
	password    string
	sessionID   string
}

func NewDexcom(account, password string,
	writers []GlucosePointsWriter, logger *zap.Logger) *DexcomClient {
	client := &DexcomClient{
		client:      http.DefaultClient,
		writers:     writers,
		logger:      logger,
		accountName: account,
		password:    password,
	}
	client.createSession()
	go client.writePeriodic()
	return client
}

func (c *DexcomClient) writePeriodic() {
	for {
		glucose, err := c.Glucose()
		if err != nil {
			c.logger.Warn("unable to get glucose points", zap.Error(err))
			continue
		}

		for _, writer := range c.writers {
			err := writer.WriteGlucosePoints(glucose)
			if err != nil {
				c.logger.Warn("unable to write glucose points to writer", zap.Error(err))
			}
		}

		dur := 5 * time.Minute
		if len(glucose) > 0 {
			dur = time.Until(glucose[0].Time.Add(5*time.Minute + 15*time.Second))
		}
		time.Sleep(dur)
	}
}

type GlucosePoint struct {
	WT    string  `json:"WT"` // Not exactly sure what this stands for.
	Value float64 `json:"Value"`
	Trend string  `json:"Trend"`
	Time  time.Time
}

func (c *DexcomClient) Glucose() ([]GlucosePoint, error) {
	// This is very rudimentary retry logic.
	// We only retry once on failure, try to recreate session again.
	glucose, err := c.glucose()
	if err == nil {
		return glucose, nil
	}

	err = c.createSession()
	if err != nil {
		return nil, err
	}
	return c.Glucose()
}

type loginRequest struct {
	AccountName   string `json:"accountName"`
	Password      string `json:"password"`
	ApplicationID string `json:"applicationId"`
}

func (c *DexcomClient) createSession() error {
	req := loginRequest{
		AccountName:   c.accountName,
		Password:      c.password,
		ApplicationID: appID,
	}
	loginUrl, _ := url.JoinPath(baseUrl, loginEndpoint)

	body, err := c.makeRequest(req, loginUrl, http.MethodPost)
	if err != nil {
		c.logger.Debug("unable to make session request", zap.Any("request", req), zap.Error(err))
		return fmt.Errorf("unable to make request: %w", err)
	}

	c.sessionID = strings.Trim(string(body), "\"")
	return nil
}

func (c *DexcomClient) glucose() ([]GlucosePoint, error) {
	params := url.Values{
		"sessionId": {c.sessionID},
		"minutes":   {minuteMax},
		"maxCount":  {countMax},
	}
	glucoseUrl, _ := url.JoinPath(baseUrl, glucoseEndpoint)
	glucoseUrl = glucoseUrl + "?" + params.Encode()

	body, err := c.makeRequest(nil, glucoseUrl, http.MethodGet)
	if err != nil {
		c.logger.Debug("unable to make glucose request", zap.Error(err))
		return nil, fmt.Errorf("unable to make request: %w", err)
	}

	var glucose []GlucosePoint
	err = json.Unmarshal(body, &glucose)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal json: %w", err)
	}

	for i, gp := range glucose {
		parsedTime := strings.Trim(gp.WT[4:], "()")
		unixMs, err := strconv.Atoi(parsedTime)
		if err != nil {
			return nil, fmt.Errorf("unable to convert to int: %w", err)
		}
		glucose[i].Time = time.UnixMilli(int64(unixMs))
	}

	return glucose, nil
}

func (c *DexcomClient) makeRequest(req any, url, method string) ([]byte, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(method, url, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("unable to execute request: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
