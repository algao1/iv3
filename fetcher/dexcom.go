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

	"github.com/algao1/iv3/store"
	"go.uber.org/zap"
)

const (
	appID           = "d89443d2-327c-4a6f-89e5-496bbb0317db"
	baseUrl         = "https://share2.dexcom.com/ShareWebServices/Services"
	loginEndpoint   = "General/LoginPublisherAccountById"
	authEndpoint    = "General/AuthenticatePublisherAccount"
	glucoseEndpoint = "Publisher/ReadPublisherLatestGlucoseValues"

	minuteMax = "1440"
	countMax  = "288"
)

type GlucosePointsWriter interface {
	WriteGlucosePoints(glucose []store.GlucosePoint) error
}

type DexcomClient struct {
	client  *http.Client
	writers []GlucosePointsWriter
	logger  *zap.Logger

	accountName string
	password    string
	accountID   string
	sessionID   string
}

func NewDexcom(account, password string,
	writers []GlucosePointsWriter, logger *zap.Logger) *DexcomClient {
	client := &DexcomClient{
		client:      &http.Client{Timeout: 5 * time.Second},
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
			c.logger.Error("unable to get glucose points", zap.Error(err))
			time.Sleep(10 * time.Second)
			continue
		}

		for _, writer := range c.writers {
			err := writer.WriteGlucosePoints(glucose)
			if err != nil {
				c.logger.Error("unable to write glucose points to writer", zap.Error(err))
			}
		}

		dur := 5 * time.Minute
		if len(glucose) > 0 {
			dur = time.Until(glucose[0].Time.Add(5*time.Minute + 10*time.Second))
		}
		time.Sleep(dur)
	}
}

func (c *DexcomClient) Glucose() ([]store.GlucosePoint, error) {
	// This is very rudimentary retry logic.
	// We only retry once on failure, try to recreate session again.
	glucose, err := c.glucose()
	if err == nil {
		return glucose, nil
	}

	err = c.createSession()
	if err != nil {
		return nil, fmt.Errorf("unable to create session: %w", err)
	}
	return c.glucose()
}

func (c *DexcomClient) getAccountId() error {
	req := authRequest{
		AccountName:   c.accountName,
		Password:      c.password,
		ApplicationID: appID,
	}
	authUrl, _ := url.JoinPath(baseUrl, authEndpoint)

	body, err := c.makeRequest(req, authUrl, http.MethodPost)
	if err != nil {
		c.logger.Warn("unable to make auth request", zap.Any("request", req), zap.Error(err))
		return fmt.Errorf("unable to make request: %w", err)
	}
	c.accountID = strings.Trim(string(body), "\"")
	c.logger.Info("succesfully got account ID", zap.String("accountID", c.accountID))

	return nil
}

func (c *DexcomClient) createSession() error {
	if c.accountID == "" {
		if err := c.getAccountId(); err != nil {
			return fmt.Errorf("unable to create session: %w", err)
		}
	}

	req := loginRequest{
		AccountID:     c.accountID,
		Password:      c.password,
		ApplicationID: appID,
	}
	loginUrl, _ := url.JoinPath(baseUrl, loginEndpoint)

	body, err := c.makeRequest(req, loginUrl, http.MethodPost)
	if err != nil {
		c.logger.Warn("unable to make session request", zap.Any("request", req), zap.Error(err))
		return fmt.Errorf("unable to make request: %w", err)
	}
	c.sessionID = strings.Trim(string(body), "\"")
	c.logger.Info("succesfully got session ID", zap.String("sessionID", c.sessionID))

	return nil
}

func (c *DexcomClient) glucose() ([]store.GlucosePoint, error) {
	params := url.Values{
		"sessionId": {c.sessionID},
		"minutes":   {minuteMax},
		"maxCount":  {countMax},
	}
	glucoseUrl, _ := url.JoinPath(baseUrl, glucoseEndpoint)
	glucoseUrl = glucoseUrl + "?" + params.Encode()

	body, err := c.makeRequest(nil, glucoseUrl, http.MethodGet)
	if err != nil {
		c.logger.Warn("unable to make glucose request", zap.Error(err))
		return nil, fmt.Errorf("unable to make request: %w", err)
	}

	var glucose []store.GlucosePoint
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

type loginRequest struct {
	AccountID     string `json:"accountId"`
	Password      string `json:"password"`
	ApplicationID string `json:"applicationId"`
}

type authRequest struct {
	AccountName   string `json:"accountName"`
	Password      string `json:"password"`
	ApplicationID string `json:"applicationId"`
}
