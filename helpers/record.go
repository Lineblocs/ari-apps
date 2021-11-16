package helpers
import (
	"context"
	"fmt"
	"encoding/json"

	"lineblocs.com/processor/api"
	"lineblocs.com/processor/types"
	"github.com/google/uuid"
	"github.com/CyCoreSystems/ari/v5"
)

type Record struct {
	Bridge *types.LineBridge
	Channel *types.LineChannel
	User *types.User
	Call *types.Call
	Handle *ari.LiveRecordingHandle
}

type RecordingParams struct {
	UserId int `json:"user_id"`
	Tag string `json:"tag"`
	Status string `json:"status"`
	WorkspaceId int `json:"workspace_id"`
}


func NewRecording(user *types.User) (*Record) {
	record := Record{
		User: user }

	return &record
}
func (r *Record) createAPIResource() (string, error) {
	user := r.User
	uniq, err := uuid.NewUUID()
	if err != nil {
		fmt.Printf("recording fail to create UUID. err: %s\r\n", err.Error())
		return "", err
	}



	id := uniq.String()
	params := RecordingParams{
		UserId: user.Id,
		Tag: "",
		Status: "started",
		WorkspaceId: user.Workspace.Id, }


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
	_ = resp.Headers.Get("x-recording-id")
	return id, nil
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
	opts := &ari.RecordingOptions{}
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
	opts := &ari.RecordingOptions{}
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
}

