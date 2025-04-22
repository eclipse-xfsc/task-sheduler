package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	goahttp "goa.design/goa/v3/http"
	goa "goa.design/goa/v3/pkg"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"golang.org/x/sync/errgroup"

	auth "github.com/eclipse-xfsc/microservice-core-go/pkg/auth"
	graceful "github.com/eclipse-xfsc/microservice-core-go/pkg/graceful"
	goahealth "github.com/eclipse-xfsc/task-sheduler/gen/health"
	goahealthsrv "github.com/eclipse-xfsc/task-sheduler/gen/http/health/server"
	goaopenapisrv "github.com/eclipse-xfsc/task-sheduler/gen/http/openapi/server"
	goatasksrv "github.com/eclipse-xfsc/task-sheduler/gen/http/task/server"
	goatasklistsrv "github.com/eclipse-xfsc/task-sheduler/gen/http/task_list/server"
	"github.com/eclipse-xfsc/task-sheduler/gen/openapi"
	goatask "github.com/eclipse-xfsc/task-sheduler/gen/task"
	goatasklist "github.com/eclipse-xfsc/task-sheduler/gen/task_list"
	"github.com/eclipse-xfsc/task-sheduler/internal/clients/cache"
	"github.com/eclipse-xfsc/task-sheduler/internal/clients/event"
	"github.com/eclipse-xfsc/task-sheduler/internal/clients/policy"
	"github.com/eclipse-xfsc/task-sheduler/internal/config"
	"github.com/eclipse-xfsc/task-sheduler/internal/executor"
	"github.com/eclipse-xfsc/task-sheduler/internal/listexecutor"
	"github.com/eclipse-xfsc/task-sheduler/internal/service"
	"github.com/eclipse-xfsc/task-sheduler/internal/service/health"
	"github.com/eclipse-xfsc/task-sheduler/internal/service/task"
	"github.com/eclipse-xfsc/task-sheduler/internal/service/tasklist"
	"github.com/eclipse-xfsc/task-sheduler/internal/storage"
)

var Version = "0.0.0+development"

func main() {
	var cfg config.Config
	if err := envconfig.Process("", &cfg); err != nil {
		log.Fatalf("cannot load configuration: %v", err)
	}

	logger, err := createLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalln(err)
	}
	defer logger.Sync() //nolint:errcheck

	logger.Info("task service started", zap.String("version", Version), zap.String("goa", goa.Version()))

	// connect to mongo db
	db, err := mongo.Connect(
		context.Background(),
		options.Client().ApplyURI(cfg.Mongo.Addr).SetAuth(options.Credential{
			Username:      cfg.Mongo.User,
			Password:      cfg.Mongo.Pass,
			AuthMechanism: cfg.Mongo.AuthMechanism,
		}),
	)
	if err != nil {
		logger.Fatal("error connecting to mongodb", zap.Error(err))
	}
	defer db.Disconnect(context.Background()) //nolint:errcheck

	// create storage
	storage := storage.New(db)

	httpClient := httpClient()

	oauthClient := httpClient
	if cfg.Auth.Enabled {
		// Create an HTTP Client which automatically issues and carries an OAuth2 token.
		// The token will auto-refresh when its expiration is near.
		oauthCtx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
		oauthClient = newOAuth2Client(oauthCtx, cfg.OAuth.ClientID, cfg.OAuth.ClientSecret, cfg.OAuth.TokenURL)
	}

	// create policy client
	policy := policy.New(cfg.Policy.Addr, oauthClient)

	// create cache client
	cache := cache.New(cfg.Cache.Addr, cache.WithHTTPClient(oauthClient))

	var events *event.Client
	if cfg.Nats.Addr != "" {
		events, err = event.New(storage, storage, cfg.Nats.Addr, cfg.Nats.Subject)
		if err != nil {
			logger.Fatal("failed to create events client", zap.Error(err))
		}
		defer events.Close(context.Background()) //nolint:errcheck
	} else {
		logger.Info("task service is not able to subscribe for cache events")
	}

	// create task executor
	executor := executor.New(
		storage,
		policy,
		storage,
		cache,
		cfg.Executor.Workers,
		cfg.Executor.PollInterval,
		cfg.Executor.MaxTaskRetries,
		httpClient,
		logger,
	)

	listExecutor := listexecutor.New(
		storage,
		policy,
		storage,
		cache,
		cfg.ListExecutor.Workers,
		cfg.ListExecutor.PollInterval,
		httpClient,
		logger,
	)

	// create services
	var (
		taskSvc     goatask.Service
		taskListSvc goatasklist.Service
		healthSvc   goahealth.Service
	)
	{
		taskSvc = task.New(storage, storage, cache, logger)
		taskListSvc = tasklist.New(storage, storage, cache, logger)
		healthSvc = health.New(Version)
	}

	// create endpoints
	var (
		taskEndpoints     *goatask.Endpoints
		taskListEndpoints *goatasklist.Endpoints
		healthEndpoints   *goahealth.Endpoints
		openapiEndpoints  *openapi.Endpoints
	)
	{
		taskEndpoints = goatask.NewEndpoints(taskSvc)
		taskListEndpoints = goatasklist.NewEndpoints(taskListSvc)
		healthEndpoints = goahealth.NewEndpoints(healthSvc)
		openapiEndpoints = openapi.NewEndpoints(nil)
	}

	// Provide the transport specific request decoder and response encoder.
	// The goa http package has built-in support for JSON, XML and gob.
	// Other encodings can be used by providing the corresponding functions,
	// see goa.design/implement/encoding.
	var (
		dec = goahttp.RequestDecoder
		enc = goahttp.ResponseEncoder
	)

	// Build the service HTTP request multiplexer and configure it to serve
	// HTTP requests to the service endpoints.
	mux := goahttp.NewMuxer()

	// Wrap the endpoints with the transport specific layers. The generated
	// server packages contains code generated from the design which maps
	// the service input and output data structures to HTTP requests and
	// responses.
	var (
		taskServer     *goatasksrv.Server
		taskListServer *goatasklistsrv.Server
		healthServer   *goahealthsrv.Server
		openapiServer  *goaopenapisrv.Server
	)
	{
		taskServer = goatasksrv.New(taskEndpoints, mux, dec, enc, nil, errFormatter)
		taskListServer = goatasklistsrv.New(taskListEndpoints, mux, dec, enc, nil, errFormatter)
		healthServer = goahealthsrv.New(healthEndpoints, mux, dec, enc, nil, errFormatter)
		openapiServer = goaopenapisrv.New(openapiEndpoints, mux, dec, enc, nil, errFormatter, nil, nil)
	}

	// Apply Authentication middleware if enabled.
	if cfg.Auth.Enabled {
		m, err := auth.NewMiddleware(cfg.Auth.JwkURL, cfg.Auth.RefreshInterval, httpClient)
		if err != nil {
			log.Fatalf("failed to create authentication middleware: %v", err)
		}
		taskServer.Use(m.Handler())
		taskListServer.Use(m.Handler())
	}

	// Configure the mux.
	goatasksrv.Mount(mux, taskServer)
	goatasklistsrv.Mount(mux, taskListServer)
	goahealthsrv.Mount(mux, healthServer)
	goaopenapisrv.Mount(mux, openapiServer)

	// expose metrics
	go exposeMetrics(cfg.Metrics.Addr, logger)

	var handler http.Handler = mux
	srv := &http.Server{
		Addr:         cfg.HTTP.Host + ":" + cfg.HTTP.Port,
		Handler:      handler,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error {
		if err := graceful.Shutdown(ctx, srv, 20*time.Second); err != nil {
			logger.Error("server shutdown error", zap.Error(err))
			return err
		}
		return errors.New("server stopped successfully")
	})
	g.Go(func() error {
		return executor.Start(ctx)
	})
	g.Go(func() error {
		return listExecutor.Start(ctx)
	})
	if events != nil {
		g.Go(func() error {
			return events.Start(ctx)
		})
	}
	if err := g.Wait(); err != nil {
		logger.Error("run group stopped", zap.Error(err))
	}

	logger.Info("bye bye")
}

func createLogger(logLevel string, opts ...zap.Option) (*zap.Logger, error) {
	var level = zapcore.InfoLevel
	if logLevel != "" {
		err := level.UnmarshalText([]byte(logLevel))
		if err != nil {
			return nil, err
		}
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(level)
	config.DisableStacktrace = true
	config.EncoderConfig.TimeKey = "ts"
	config.EncoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
	return config.Build(opts...)
}

func errFormatter(ctx context.Context, e error) goahttp.Statuser {
	return service.NewErrorResponse(ctx, e)
}

func httpClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).DialContext,
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			TLSHandshakeTimeout: 10 * time.Second,
			IdleConnTimeout:     60 * time.Second,
		},
		Timeout: 20 * time.Second,
	}
}

func newOAuth2Client(ctx context.Context, cID, cSecret, tokenURL string) *http.Client {
	oauthCfg := clientcredentials.Config{
		ClientID:     cID,
		ClientSecret: cSecret,
		TokenURL:     tokenURL,
	}

	return oauthCfg.Client(ctx)
}

func exposeMetrics(addr string, logger *zap.Logger) {
	promMux := http.NewServeMux()
	promMux.Handle("/metrics", promhttp.Handler())
	logger.Info(fmt.Sprintf("exposing prometheus metrics at %s/metrics", addr))
	if err := http.ListenAndServe(addr, promMux); err != nil { //nolint:gosec
		logger.Error("error exposing prometheus metrics", zap.Error(err))
	}
}
