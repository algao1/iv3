package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/algao1/iv3/config"
	"github.com/algao1/iv3/fetcher"
	"go.uber.org/zap"
)

type PointsReadWriter interface {
	ReadGlucosePoints(startTs, endTs int) ([]fetcher.GlucosePoint, error)
	ReadInsulinPoints(startTs, endTs int) ([]fetcher.InsulinPoint, error)
	WriteInsulinPoint(point fetcher.InsulinPoint) error
}

type HttpServer struct {
	username string
	password string

	readWriter PointsReadWriter
	insulin    []config.InsulinConfig

	logger *zap.Logger
}

func NewHttpServer(username, password string,
	readWriter PointsReadWriter, logger *zap.Logger) *HttpServer {
	return &HttpServer{
		username:   username,
		password:   password,
		readWriter: readWriter,
		logger:     logger,
	}
}

func (s *HttpServer) RegisterInsulin(insulin []config.InsulinConfig) {
	s.insulin = insulin
}

func (s *HttpServer) Serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/glucose", s.basicAuth(s.getGlucoseHandler))
	mux.HandleFunc("/insulin", s.basicAuth(s.getInsulinHandler))
	mux.HandleFunc("/insulin/write", s.basicAuth(s.writeInsulinHandler))
	mux.HandleFunc("/insulinTypes", s.basicAuth(s.getInsulinTypesHandler))

	srv := &http.Server{
		Addr:         ":443",
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	s.logger.Info("starting server", zap.String("addr", srv.Addr))
	err := srv.ListenAndServeTLS("certfile.crt", "keyfile.key")
	s.logger.Fatal("unable to listen and serve TLS", zap.Error(err))
}

func (s *HttpServer) getGlucoseHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("got GET request for /glucose", zap.Any("query", r.URL.Query()))

	startTs, endTs, err := getStartEndTs(r.URL.Query())
	if err != nil {
		fmt.Fprintln(w, "unable to parse start/end timestamps: %w", err)
		return
	}

	glucose, err := s.readWriter.ReadGlucosePoints(startTs, endTs)
	if err != nil {
		fmt.Fprintln(w, "unable to fetch glucose: %w", err)
		return
	}
	json.NewEncoder(w).Encode(glucose)
}

func (s *HttpServer) getInsulinHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("got GET request for /insulin", zap.Any("query", r.URL.Query()))

	startTs, endTs, err := getStartEndTs(r.URL.Query())
	if err != nil {
		fmt.Fprintln(w, "unable to parse start/end timestamps: %w", err)
		return
	}

	insulin, err := s.readWriter.ReadInsulinPoints(startTs, endTs)
	if err != nil {
		fmt.Fprintln(w, "unable to fetch insulin: %w", err)
		return
	}
	json.NewEncoder(w).Encode(insulin)
}

type intermedInsulinPoint struct {
	Value int    `json:"value"`
	Type  string `json:"type"`
	Ts    int    `json:"ts"`
}

func (s *HttpServer) writeInsulinHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("got POST request for /insulin/write", zap.Any("query", r.URL.Query()))

	var intPoint intermedInsulinPoint
	err := json.NewDecoder(r.Body).Decode(&intPoint)
	if err != nil {
		fmt.Fprintln(w, "unable to decode insulin point: %w", err)
		return
	}

	point := fetcher.InsulinPoint{
		Value: intPoint.Value,
		Type:  intPoint.Type,
		Time:  time.Unix(int64(intPoint.Ts), 0),
	}
	err = s.readWriter.WriteInsulinPoint(point)
	if err != nil {
		fmt.Fprintln(w, "unable to write insulin point: %w", err)
		return
	}
}

func (s *HttpServer) getInsulinTypesHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("got GET request for /insulinTypes", zap.Any("query", r.URL.Query()))
	json.NewEncoder(w).Encode(s.insulin)
}

func getStartEndTs(values url.Values) (int, int, error) {
	startStr := values.Get("start")
	if startStr == "" {
		return 0, 0, fmt.Errorf("no start timestamp provided")
	}
	startTs, err := strconv.Atoi(startStr)
	if err != nil {
		return 0, 0, fmt.Errorf("start timestamp is not int: %w", err)
	}

	endStr := values.Get("end")
	if endStr == "" {
		return 0, 0, fmt.Errorf("no end timestamp provided")
	}
	endTs, err := strconv.Atoi(endStr)
	if err != nil {
		return 0, 0, fmt.Errorf("end timestamp is not int: %w", err)
	}

	return startTs, endTs, nil
}

// This is some very basic authentication, I think it is ok for now.
// If I need something better, I can always swap out the middleware.
func (s *HttpServer) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(s.username))
			expectedPasswordHash := sha256.Sum256([]byte(s.password))

			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)

			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}
