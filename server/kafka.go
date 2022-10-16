package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/davidtrse/graceful/kafkas"
	"github.com/davidtrse/graceful/pkg/app"
	"github.com/labstack/gommon/log"
)

func Kafka() {

	InitKafka()
	go mainLoop()

	// catch the signal
	existOSsignal := make(chan os.Signal, 1)
	signal.Notify(existOSsignal, syscall.SIGINT, syscall.SIGTERM)

	wait := make(chan bool, 1)
d:
	for {
		select {
		case <-existOSsignal:
			wait <- true
			fmt.Println("shutting down...")
			app.Instance.KafkaManager.Close()
			app.Instance.KafkaManager.StopReadMessage()
		case <-wait:
			if app.Instance.KafkaManager.IsDone() {
				break d
			} else {
				wait <- true
			}
		}
		time.Sleep(1 * time.Second)
	}

	app.Instance.KafkaManager.StopWriteMessage()
	fmt.Println("Server exited.")
}

func InitKafka() {
	km, err := kafkas.NewKafkaManager(&kafkas.KafkaConfig{
		Hosts:   "127.0.0.1:9092",
		GroupId: "vodtrans",
		Topics:  "topic-test",
	})

	km.Context = context.Background()
	if err != nil {
		log.Fatalf("Failed to create Kafka manager: %s", err.Error())
	}

	km.CreateReader()
	app.Instance = &app.Context{
		KafkaManager: km,
	}
}

func mainLoop() {
	for {
		msg, err := app.Instance.KafkaManager.ReadMessage("")
		if err != nil {
			if err == io.EOF {
				fmt.Println("Read message error. Reader is closed")
				break
			}

			if errors.Is(err, kafkas.ErrContextClosed) {
				fmt.Println("Context done! Reader is closed")
				break
			}

			log.Error("Failed on reading msg")
			continue
		}
		app.Instance.KafkaManager.StartNewTranscode(string(msg.Value))
		// if receive message is "Slow", will sleep 30 second while loop and print 0-29
		if string(msg.Value) == "Slow" {
			fmt.Println("Slow.....")

			for i := 0; i < 30; i++ {
				fmt.Printf("Slow: %d\n", i)
				time.Sleep(1 * time.Second)
			}
		} else {
			fmt.Printf("Message: msg=%s \n", string(msg.Value))
			for i := 0; i < 2; i++ {
				fmt.Printf("Fast: %d\n", i)
				time.Sleep(1 * time.Second)
			}
			if kafkas.IsNotEmpty(msg) {
				fmt.Println("msg not empty..")
			}
		}
		app.Instance.KafkaManager.DoneTranscode(string(msg.Value))
	}
}
