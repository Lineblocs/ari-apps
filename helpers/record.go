package helpers
import (
	"context"
	"fmt"
	"os"
	"strconv"
	"encoding/json"

	"lineblocs.com/processor/api"
	"lineblocs.com/processor/types"
	"github.com/google/uuid"
	"github.com/CyCoreSystems/ari/v5"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

type Record struct {
	Bridge *types.LineBridge
	Channel *types.LineChannel
	User *types.User
	StorageServer *types.StorageServer
	CallId *int
	Handle *ari.LiveRecordingHandle
	EventProducer *kafka.Producer
	Id string
	Trim bool	
}

type RecordingParams struct {
	Id int `json:"id"`
	UserId int `json:"user_id"`
	CallId *int `json:"call_id"`
	Tag string `json:"tag"`
	Status string `json:"status"`
	WorkspaceId int `json:"workspace_id"`
	StorageId string `json:"storage_id"`
	StorageServerIp string `json:"storage_server_ip"`
	Trim bool `json:"trim"`
}


func NewRecording(server *types.StorageServer, producer *kafka.Producer, user *types.User, callId *int, trim bool) (*Record) {
	record := Record{
		StorageServer: server,
		User: user, 
		CallId:callId, 
		Trim: trim,
		EventProducer: producer,
	 }

	return &record
}
func (r *Record) createAPIResource() (string, error) {
	user := r.User
	callId := r.CallId
	uniq, err := uuid.NewUUID()
	if err != nil {
		fmt.Printf("recording fail to create UUID. err: %s\r\n", err.Error())
		return "", err
	}


	id := uniq.String()
	params := RecordingParams{
		UserId: user.Id,
		CallId:callId,
		Tag: "",
		Status: "started",
		WorkspaceId: user.Workspace.Id,
		Trim: r.Trim,
		StorageId: id,
		StorageServerIp: r.StorageServer.Ip}


	body, err := json.Marshal( params )
	if err != nil {
		fmt.Printf( "error occured: %s\r\n", err.Error() )
		return "", err
	}

	fmt.Println("creating recording call...")
	resp, err := api.SendHttpRequest( "/recording/createRecording", body )
	if err != nil {
		fmt.Printf( "error occured: %s\r\n", err.Error() )
		return "", err
	}

	r.Id = resp.Headers.Get("x-recording-id")

	return r.Id, nil
}


func (r *Record) InitiateRecordingForBridge(bridge *types.LineBridge) (string, error) {
	r.Bridge = bridge 
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	id, err := r.createAPIResource()
	if err != nil {
		fmt.Printf("failed to record. err: %s\r\n", err.Error())
		return "", err
	}

	key := ari.NewKey(ari.LiveRecordingKey, id)
	opts := &ari.RecordingOptions{ Format: "wav" }
	hndl, err := bridge.Bridge.Record(key.ID, opts)
	if err != nil {
		fmt.Printf("failed to record. err: %s\r\n", err.Error())
		return "",err
	}
	r.Handle = hndl
	return id, nil
}

func (r *Record) InitiateRecordingForChannel(channel *types.LineChannel) (string, error) {
	r.Channel = channel
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	id, err := r.createAPIResource()
	if err != nil {
		fmt.Printf("failed to record. err: %s", err.Error())
		return "",err
	}

	key := ari.NewKey(ari.LiveRecordingKey, id)
	opts := &ari.RecordingOptions{ Format: "wav" }
	hndl, err := channel.Channel.Record(key.ID, opts)
	if err != nil {
		fmt.Printf("failed to record. err: %s\r\n", err.Error())
		return "",err
	}


	r.Handle = hndl

	return id, nil
}


func (r *Record) Stop() {
	r.Handle.Stop()

	recordingId, err := strconv.Atoi(r.Id)
    if err != nil {
		fmt.Printf( "error occured while setting recording status: %s\r\n", err.Error() )
		return
    }

	producer := r.EventProducer

	params := RecordingParams{
		Id: recordingId,
		Status: "completed",
	}


	body, err := json.Marshal( params )
	if err != nil {
		fmt.Printf( "error occured while setting recording status: %s\r\n", err.Error() )
		return;
	}

	_, err = api.SendHttpRequest( "/recording/setRecordingStatus", body )
	if err != nil {
		fmt.Printf( "error occured while setting recording status: %s\r\n", err.Error() )
	}

	// generate an event to notify other services that recording is ready for processing
	if producer != nil {
		fmt.Println( "notifying all services that recording is complete and ready for further processing" )
		topic := os.Getenv("KAFKA_RECORDINGS_TOPIC")
		recordingData := struct {
			RecordingId  int `json:"id"`
			Status string `json:"status"`
		}{
			RecordingId: recordingId,
			Status: "complete",
		}

		// Marshal the struct into JSON
		recordingMsg, err := json.Marshal(recordingData)
		if err != nil {
			fmt.Println("Error marshalling JSON:", err)
			return
		}
		err = producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value: []byte(recordingMsg)},
			nil, // delivery channel
		)
	}
}

