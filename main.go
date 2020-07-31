package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/gertd/goproxy/pkg/version"
	"github.com/goproxy/goproxy"
	"github.com/goproxy/goproxy/cacher"
)

type key int

const (
	appName          = "goproxy"
	requestIDKey key = 0
)

var (
	addr     string
	cacheDir string
	verInfo  bool
	healthy  int32
)

func main() {
	flag.StringVar(&addr, "addr", ":8080", "server listen address")
	flag.StringVar(&cacheDir, "cache", "~/.cache/goproxy", "cache directory")
	flag.BoolVar(&verInfo, "version", false, "version info")
	flag.Parse()

	if verInfo {
		fmt.Printf("%s - %s\n", appName, version.GetInfo())
		os.Exit(0)
	}

	logger := log.New(os.Stdout, "http: ", log.LstdFlags)
	logger.Println("Server is starting...")

	router := http.NewServeMux()
	router.Handle("/healthz", healthz())

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	server := &http.Server{
		Addr:         addr,
		Handler:      tracing(nextRequestID)(logging(logger)(proxy(cacheDir))),
		ErrorLog:     logger,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)

	go func() {
		<-quit
		logger.Println("Server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		server.SetKeepAlivesEnabled(false)
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	logger.Println("Server is ready to handle requests at", addr)
	atomic.StoreInt32(&healthy, 1)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("Could not listen on %s: %v\n", addr, err)
	}

	<-done
	logger.Println("Server stopped")
}

func proxy(cachedir string) http.Handler {
	proxy := goproxy.New()
	proxy.Cacher = &cacher.Disk{Root: cachedir}

	return proxy
}

func healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}

func logging(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				logger.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
