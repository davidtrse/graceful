package tus

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/davidtrse/graceful/log"
	"github.com/davidtrse/graceful/pkg/app"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/credentials"
)

const (
	dirPath = "./upload"
	dirName = "tusSave"
)

var (
	tracer = otel.Tracer("tus")
)

func Run() {
	Init()
	shutdown := configureStdout(context.Background())
	defer shutdown()
	log.Println("TUS Server started")
	// Create a new FileStore instance which is responsible for
	// storing the uploaded file on disk in the specified directory.
	// This path _must_ exist before tusd will store uploads in it.
	// If you want to save them on a different medium, for example
	// a remote FTP server, you can implement your own storage backend
	// by implementing the tusd.DataStore interface.
	store := filestore.FileStore{
		Path: dirPath,
	}

	// A storage backend for tusd may consist of multiple different parts which
	// handle upload creation, locking, termination and so on. The composer is a
	// place where all those separated pieces are joined together. In this example
	// we only use the file store but you may plug in multiple.
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	// Create a new HTTP handler for the tusd server by providing a configuration.
	// The StoreComposer property must be set to allow the handler to function.
	handler, err := tusd.NewHandler(tusd.Config{
		BasePath:              "/files/",
		StoreComposer:         composer,
		NotifyCompleteUploads: true,
		NotifyCreatedUploads:  true,
		PreUploadCreateCallback: func(hook tusd.HookEvent) error {
			fmt.Println("PreUploadCreateCallback")
			fmt.Println("PreUploadCreateCallback:  IsAcceptingRequestStopped ====>", app.Instance.GracefulTUSManager.IsReceivingRequest())
			if !app.Instance.GracefulTUSManager.IsReceivingRequest() {
				return tusd.NewHTTPError(errors.New("server not available"), http.StatusServiceUnavailable)
			}
			return nil
		},
		PreFinishResponseCallback: func(hook tusd.HookEvent) error {
			fmt.Println("Pre-finish create handler success")

			return nil
		},
	})
	if err != nil {
		panic(fmt.Errorf("Unable to create handler: %s", err))
	}

	/// Start another goroutine for receiving events from the handler whenever
	// an upload is created.
	go func() {
		for {
			event := <-handler.CreatedUploads
			fmt.Printf("Upload %s created\n", event.Upload.ID)
			app.Instance.GracefulTUSManager.StartNewUpload(event.Upload.ID)
		}
	}()

	// Start another goroutine for receiving events from the handler whenever
	// an upload is completed. The event will contains details about the upload
	// itself and the relevant HTTP request.
	go func() {
		for {
			event := <-handler.CompleteUploads
			fmt.Printf("Upload %s finished\n", event.Upload.ID)
			app.Instance.GracefulTUSManager.DoneUpload(event.Upload.ID)
		}
	}()

	e := echo.New()
	app.Instance.GracefulTUSManager.StartReceivingRequest()
	e.Use(app.Instance.GracefulTUSManager.EchoMiddleware())

	cors := middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		MaxAge:       3600,
	})
	e.Use(cors)

	e.POST("/files", echo.WrapHandler(http.HandlerFunc(handler.PostFile)), echo.WrapMiddleware(tusmiddleware))
	e.HEAD("/files/:fileID", echo.WrapHandler(http.HandlerFunc(handler.HeadFile)), echo.WrapMiddleware(tusmiddleware))
	e.PATCH("/files/:fileID", echo.WrapHandler(http.HandlerFunc(handler.PatchFile)), echo.WrapMiddleware(tusmiddleware))
	e.GET("/files/:fileID", echo.WrapHandler(http.HandlerFunc(handler.GetFile)))
	e.GET("/", hello)
	e.GET("/l", helloSlow)
	// Start server in dedicated go routine
	go func() {
		if err := e.Start(":8180"); err != nil {
			if err != nil && err != http.ErrServerClosed {
				e.Logger.Errorf("shutting down the server..., err=%s", err)
			} else {
				e.Logger.Info("Server terminated...")
			}
		}
	}()

	// GRACEFUL SHUTDOWN
	// Wait for interrupt signal to gracefully shutdown the server.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	isOSExist := make(chan os.Signal, 1)
	signal.Notify(
		isOSExist,
		syscall.SIGHUP,  // kill - SIGHUP XXXX
		syscall.SIGINT,  // kill - SIGIN XXXX or Ctrl + C
		syscall.SIGQUIT, // kill - SIGQUICT XXXX
	)
	isOk := make(chan bool, 1)
d:
	for {
		select {
		case <-isOSExist:
			isOk <- true
			fmt.Println("====> Graceful shutting down...")
			app.Instance.GracefulTUSManager.StopReceivingRequest()
		case <-isOk:
			if app.Instance.GracefulTUSManager.CanShutdown() {
				fmt.Println("====> GracefulTUSManager.IsDone=true...")
				break d
			} else {
				isOk <- true
				fmt.Println("====> Graceful shutting down...")
				time.Sleep(1 * time.Second)
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	} else {
		fmt.Println("===> Server exited graceful.")
	}
}

func Init() {
	app.Instance = &app.Context{
		GracefulTUSManager: app.NewShutdownManage(),
	}
}

func hello(c echo.Context) error {
	// Each execution of the run loop, we should get a new "root" span and context.
	ctx, span := tracer.Start(c.Request().Context(), "hello", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	log.Infoff(ctx, "hello spanId=%s", span.SpanContext().SpanID())
	log.Infoff(ctx, "hello span_id=%s", span.SpanContext().SpanID())
	log.Infoff(ctx, "hello trace_id=%s", span.SpanContext().TraceID())

	bag := baggage.FromContext(ctx)

	serverAttribute := attribute.String("server-attribute", "foo")
	var baggageAttributes []attribute.KeyValue
	baggageAttributes = append(baggageAttributes, serverAttribute)
	for _, member := range bag.Members() {
		baggageAttributes = append(baggageAttributes, attribute.String("baggage key:"+member.Key(), member.Value()))
	}
	span.SetAttributes(baggageAttributes...)

	c.Response().Write([]byte(strconv.Itoa(r1.Intn(100))))
	return nil
}

func helloSlow(echo echo.Context) error {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	for i := 0; i < 15; i++ {
		fmt.Println(i)
		time.Sleep(1 * time.Second)
	}
	echo.Response().Write([]byte(strconv.Itoa(r1.Intn(100))))
	return nil
}

func tusmiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow overriding the HTTP method. The reason for this is
		// that some libraries/environments to not support PATCH and
		// DELETE requests, e.g. Flash in a browser and parts of Java
		if newMethod := r.Header.Get("X-HTTP-Method-Override"); newMethod != "" {
			r.Method = newMethod
		}

		header := w.Header()

		if origin := r.Header.Get("Origin"); origin != "" {
			header.Set("Access-Control-Allow-Origin", origin)

			if r.Method == "OPTIONS" {
				allowedMethods := "POST, HEAD, PATCH, OPTIONS"

				// Preflight request
				header.Add("Access-Control-Allow-Methods", allowedMethods)
				header.Add("Access-Control-Allow-Headers", "Authorization, Origin, X-Requested-With, X-Request-ID, X-HTTP-Method-Override, Content-Type, Upload-Length, Upload-Offset, Tus-Resumable, Upload-Metadata, Upload-Defer-Length, Upload-Concat")
				header.Set("Access-Control-Max-Age", "86400")

			} else {
				// Actual request
				header.Add("Access-Control-Expose-Headers", "Upload-Offset, Location, Upload-Length, Tus-Version, Tus-Resumable, Tus-Max-Size, Tus-Extension, Upload-Metadata, Upload-Defer-Length, Upload-Concat")
			}
		}

		// Set current version used by the server
		header.Set("Tus-Resumable", "1.0.0")

		// Add nosniff to all responses https://golang.org/src/net/http/server.go#L1429
		header.Set("X-Content-Type-Options", "nosniff")

		// Set appropriated headers in case of OPTIONS method allowing protocol
		// discovery and end with an 204 No Content
		if r.Method == "OPTIONS" {
			header.Add("Access-Control-Allow-Origins", "*")
		}

		// Test if the version sent by the client is supported
		// GET and HEAD methods are not checked since a browser may visit this URL and does
		// not include this header. GET requests are not part of the specification.
		if r.Method != "GET" && r.Method != "HEAD" && r.Header.Get("Tus-Resumable") != "1.0.0" {
			return
		}

		// Proceed with routing the request
		h.ServeHTTP(w, r)
	})
}

func configureStdout(ctx context.Context) func() {
	configGrpcInsecure := false
	secureOption := otlptracegrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, ""))
	if configGrpcInsecure {
		secureOption = otlptracegrpc.WithInsecure()
	}

	configCollectorURL := "tempo:4317"
	exporter, _ := otlptrace.New(
		context.Background(),
		otlptracegrpc.NewClient(
			secureOption,
			otlptracegrpc.WithEndpoint(configCollectorURL),
		),
	)

	// span processor
	exp, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())

	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(exp)
	// trace provider
	configName := "gracefulService"
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(configName),
		)),
		sdktrace.WithSpanProcessor(batchSpanProcessor),
		sdktrace.WithBatcher(exporter),
	)
	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	otel.SetTracerProvider(provider)

	return func() {
		if err := provider.Shutdown(ctx); err != nil {
			panic(err)
		}
	}
}
