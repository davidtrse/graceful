package server

import (
	"context"
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

func Start() {
	quit := make(chan bool, 1)
	isTranscodeDone := make(chan bool, 1)
	keepRunning := make(chan bool, 1)
	InitKafka()
	go mainLoop(isTranscodeDone, keepRunning)

	// catch the signal
	existOSsignal := make(chan os.Signal, 1)
	signal.Notify(existOSsignal, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	wait := make(chan bool, 1)
d:
	for {
		select {
		case <-existOSsignal:
			wait <- true
			fmt.Println("shutting down...")
			app.Instance.KafkaManager.StopConsumer()
		case <-wait:
			select {
			case <-isTranscodeDone:
				fmt.Println("isTranscodeDone=true...")
				quit <- true
				break d
			case <-ctx.Done():
				quit <- true
				fmt.Println("ctx.Done=true...")
				break d
			case <-keepRunning:
				fmt.Println("keepRunning=true...")
				keepRunning <- true
				wait <- true
			default:
				fmt.Println("wait=true...")
				quit <- true
				break d
			}
		default:
			fmt.Println("inprogress...")
		}
		time.Sleep(1 * time.Second)
	}

	<-quit
	fmt.Println("Quit")
	cancel()
	app.Instance.KafkaManager.StopWriteMessage()
}

func InitKafka() {
	km, err := kafkas.NewKafkaManager(&kafkas.KafkaConfig{
		Hosts:   "127.0.0.1:9092",
		GroupId: "vodtrans",
		Topics:  "topic-test",
	})

	if err != nil {
		log.Fatalf("Failed to create Kafka manager: %s", err.Error())
	}

	km.CreateReader()
	app.Instance = &app.Context{
		KafkaManager: km,
	}
}

func mainLoop(isTranscodeDone chan bool, keepRunning chan bool) {
	for {
		msg, err := app.Instance.KafkaManager.ReadMessage("")
		if err != nil {
			if err == io.EOF {
				fmt.Println("Read message error. Reader is closed")
				break
			}

			log.Error("Failed on reading msg")
			continue
		}

		// if receive message is "Big", will sleep 5 second while loop and print 1-5
		if string(msg.Value) == "Big" {
			keepRunning <- true
			fmt.Println("Big.....")
			for i := 0; i < 5; i++ {
				fmt.Println(i)
				time.Sleep(1 * time.Second)
			}
			isTranscodeDone <- true
		} else {
			fmt.Printf("Message: msg=%s \n", string(msg.Value))
			if kafkas.IsNotEmpty(msg) {
				fmt.Println("msg not empty..")
			}
		}
	}
}
