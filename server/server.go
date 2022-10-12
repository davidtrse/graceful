package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

func Run() {
	delayTime := 10

	quit := make(chan bool, 1)
	tusDone := make(chan bool, 1)
	// Wait for interrupt signal to gracefully shutdown the server with a timeout of 10 seconds.
	// Use a buffered channel to avoid missing signals as recommended for signal.Notify
	isOSExist := make(chan os.Signal, 1)
	// go func(isOSExist chan os.Signal) {
	signal.Notify(isOSExist, os.Interrupt)
	// }(isOSExist)

	// Setup
	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	e.GET("/", func(c echo.Context) error {
		// Printout each second
		go func(tusDone chan bool) {
			for i := 0; i < delayTime+4; i++ {
				fmt.Println(i)
				time.Sleep(1 * time.Second)
			}

			tusDone <- true
		}(tusDone)
		// END printout each second

		time.Sleep(time.Duration(delayTime) * time.Second)
		fmt.Println("OK")
		return c.JSON(http.StatusOK, "OK")
	})

	// Start server
	go func() {
		if err := e.Start(":8180"); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal(err)
		}
	}()

	go func() {
		isOk := make(chan bool, 1)
		for {
			select {
			case <-isOSExist:
				isOk <- true
			case <-isOk:
				select {
				case <-tusDone:
					fmt.Println("tusDone")
					quit <- true
				default:
					fmt.Println("isOSExist")
				}
				isOk <- true
			default:
				fmt.Println("inprogress...")
			}

			time.Sleep(1 * time.Second)
		}
	}()

	<-quit
	fmt.Println("Shutting down server...")
	ctx, cancel := context.WithCancel(context.Background())
	// ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	fmt.Println("Server existed.")
}
