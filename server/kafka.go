package server

import (
	"fmt"

	"github.com/davidtrse/graceful/kafkas"
	"github.com/davidtrse/graceful/pkg/app"
	"github.com/labstack/gommon/log"
)

func Start() {
	InitKafka()
	mainLoop()
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

func mainLoop() {
	for {
		msg, err := app.Instance.KafkaManager.ReadMessage("")
		if err != nil {
			log.Error("Failed on reading msg")
			continue
		}
		if kafkas.IsNotEmpty(msg) {
			fmt.Println(msg)
		}
	}
}
