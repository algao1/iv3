package server

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/algao1/iv3/config"
	"github.com/algao1/iv3/fetcher"
	"go.uber.org/zap"
)

type GlucosePointsReader interface {
	ReadGlucosePoints(startTs, endTs int) ([]fetcher.GlucosePoint, error)
}

type HttpServer struct {
	username string
	password string

	reader  GlucosePointsReader
	insulin []config.InsulinConfig

	logger *zap.Logger
}

func NewHttpServer(username, password string,
	reader GlucosePointsReader, logger *zap.Logger) *HttpServer {
	return &HttpServer{
		username: username,
		password: password,
		reader:   reader,
		logger:   logger,
	}
}

func (s *HttpServer) RegisterInsulin(insulin []config.InsulinConfig) {
	s.insulin = insulin
}

func (s *HttpServer) Serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/glucose", s.basicAuth(s.getGlucoseHandler))
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

	startStr := r.URL.Query().Get("start")
	if startStr == "" {
		fmt.Fprintln(w, "no start timestamp provided")
		return
	}
	startTs, err := strconv.Atoi(startStr)
	if err != nil {
		fmt.Fprintln(w, "start timestamp is not int: %w", err)
		return
	}

	endStr := r.URL.Query().Get("end")
	if endStr == "" {
		fmt.Fprintln(w, "no end timestamp provided")
		return
	}
	endTs, err := strconv.Atoi(endStr)
	if err != nil {
		fmt.Fprintln(w, "end timestamp is not int: %w", err)
		return
	}

	glucose, err := s.reader.ReadGlucosePoints(startTs, endTs)
	if err != nil {
		fmt.Fprintln(w, "unable to fetch glucose: %w", err)
		return
	}
	json.NewEncoder(w).Encode(glucose)
}

func (s *HttpServer) getInsulinTypesHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("got GET request for /insulin", zap.Any("query", r.URL.Query()))
	json.NewEncoder(w).Encode(s.insulin)
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
