package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	p2mobile "github.com/MoonSHRD/p2chat-android/pkg"
)

type topicsSlice []string

func (t *topicsSlice) String() string {
	var strArr []string
	for _, str := range *t {
		strArr = append(strArr, str)
	}
	return fmt.Sprintf("%s", strArr)
}

func (t *topicsSlice) Set(value string) error {
	*t = append(*t, value)
	return nil
}

type nodeConfig struct {
	listenPort int
	matrixID   string
	topics     topicsSlice
}

func main() {
	p2mobile.Start("moonshard", "/moonshard/1.0.0", "", 0)
	cfg := parseFlags()
	p2mobile.SetMatrixID(cfg.matrixID)
	log.Println("Node started!")
	for _, str := range cfg.topics {
		log.Println("Subscribing to topic " + str)
		p2mobile.SubscribeToTopic(str)
	}
	log.Println("Node subscribed to given topics!")
	go publishTestMessageToAllTopics(*cfg)
	for {
		match := p2mobile.GetNewMatch()
		message := p2mobile.GetMessages()
		if match != "" {
			log.Println(match)
		}
		if message != "" {
			log.Println(message)
		}
		time.Sleep(2 * time.Second)
	}
}

func publishTestMessageToAllTopics(cfg nodeConfig) {
	for {
		for _, topic := range cfg.topics {
			p2mobile.PublishMessage(topic, "Test!")
		}
		time.Sleep(2 * time.Second)
	}
}

func parseFlags() *nodeConfig {
	c := &nodeConfig{}

	flag.StringVar(&c.matrixID, "matrixID", "0", "The bootstrap node wrapped_host listen address\n")
	flag.Var(&c.topics, "topic", "Topic for subscription (you can specify multiple topics, for example, -topic one -topic two")
	flag.IntVar(&c.listenPort, "port", 0, "Node listen port")

	flag.Parse()
	return c
}
