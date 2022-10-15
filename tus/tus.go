package tus

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"

	"github.com/tus/tusd/pkg/filestore"
	tusd "github.com/tus/tusd/pkg/handler"
)

const (
	dirPath = "./upload"
	dirName = "tusSave"
)

func Run() {
	isTUSDone := make(chan bool, 1)
	isTUSProcessing := make(chan bool, 1)

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
		PreUploadCreateCallback: func(hook tusd.HookEvent) error {
			isTUSProcessing <- true
			return nil
		},
		PreFinishResponseCallback: func(hook tusd.HookEvent) error {
			fmt.Println("Pre-create create handler success")
			return nil
		},
	})
	if err != nil {
		panic(fmt.Errorf("Unable to create handler: %s", err))
	}

	// Start another goroutine for receiving events from the handler whenever
	// an upload is completed. The event will contains details about the upload
	// itself and the relevant HTTP request.
	go func() {
		for {
			event := <-handler.CompleteUploads
			fmt.Printf("Upload %s finished\n", event.Upload.ID)
			isTUSDone <- true
		}
	}()

	e := echo.New()

	cors := middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		// AllowMethods: []string{
		// 	http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete,
		// 	http.MethodPatch, http.MethodHead, http.MethodOptions},
		// AllowHeaders: []string{
		// 	echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept,
		// 	echo.HeaderAuthorization, echo.HeaderAccessControlExposeHeaders,
		// 	"Location", "X-Requested-With", "X-Request-ID",
		// 	"X-HTTP-Method-Override", "Upload-Defer-Length",
		// 	"Tus-Resumable", "Tus-Max-Size", "Tus-Extension",
		// 	"upload-length", "upload-metadata", "upload-offset",
		// 	"upload-concat", "Redirect"},
		// ExposeHeaders: []string{
		// 	"Upload-Offset", "Location", "Upload-Length", "Tus-Version",
		// 	"Tus-Resumable", "Tus-Max-Size", "Tus-Extension", "Upload-Metadata",
		// 	"Upload-Defer-Length", "Upload-Concat"},
		MaxAge: 3600,
	})
	e.Use(cors)

	e.OPTIONS("/files/:fileID", echo.WrapHandler(http.HandlerFunc(handler.ServeHTTP)), echo.WrapMiddleware(tusmiddleware))
	e.POST("/files", echo.WrapHandler(http.HandlerFunc(handler.PostFile)), echo.WrapMiddleware(tusmiddleware))
	e.HEAD("/files/:fileID", echo.WrapHandler(http.HandlerFunc(handler.HeadFile)), echo.WrapMiddleware(tusmiddleware))
	e.PATCH("/files/:fileID", echo.WrapHandler(http.HandlerFunc(handler.PatchFile)), echo.WrapMiddleware(tusmiddleware))
	e.GET("/files/:fileID", echo.WrapHandler(http.HandlerFunc(handler.GetFile)))
	e.GET("/", hello)
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
	quitChan := make(chan bool, 1)
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
		case <-isOk:
			select {
			case <-isTUSProcessing:
				select {
				case <-isTUSDone:
					fmt.Println("isTUSDone=true")
					quitChan <- true
					break d
				default:
					isTUSProcessing <- true
					isOk <- true
				}
			default:
				fmt.Println("isOSExisting=true")
				quitChan <- true
				break d
			}
		case a, ok := <-isTUSProcessing:
			if ok {
				fmt.Println("isTUSProcessing", a)
			}
		case a, ok := <-isTUSDone:
			if ok {
				fmt.Println("isTUSDone", a)
			}
		default:
			fmt.Println("inprogress...")
		}

		time.Sleep(1 * time.Second)
	}

	<-quitChan
	fmt.Println("====> Graceful shutdowning...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	} else {
		fmt.Println("===> Server exited graceful.")
	}
}

func hello(echo echo.Context) error {
	for i := 0; i < 10; i++ {
		fmt.Println(i)
		time.Sleep(1 * time.Second)
	}
	echo.Response().Write([]byte("Ok"))
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
