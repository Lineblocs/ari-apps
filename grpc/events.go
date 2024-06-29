package grpc;

import (
	"github.com/confluentinc/confluent-kafka-go/kafka"
)


type ClientEvent struct {
	ClientId string `json:"client_id"`
	Type string `json:"type"`
	Data map[string]string `json:"data"`
}
type EventRegistry struct {
	WSChan chan *ClientEvent
	ClientId string `json:"client_id"`
}

var wsEventStreams = []*EventRegistry{}
var kafkaEventProducer *kafka.Producer

func createWSChan(id string) (chan *ClientEvent) {
	wsChan := make( chan *ClientEvent )
	item := EventRegistry{
		WSChan: wsChan,
		ClientId: id }
	wsEventStreams = append(wsEventStreams,&item)
	return wsChan
}

func lookupWSChan(id string) (chan *ClientEvent) {
	for _, item := range wsEventStreams {
		if item.ClientId == id {
			return item.WSChan
		}
	}
	return nil
}

func getProducer() (*kafka.Producer) {
	return kafkaEventProducer
}