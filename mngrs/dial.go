package mngrs
import (
	//"context"
	"strconv"
	"sync"
	"time"
	"context"
	"encoding/json"
	"github.com/CyCoreSystems/ari/v5"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/api"
	"lineblocs.com/processor/helpers"
)
type DialManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func (man *DialManager) manageOutboundCallLeg(outboundChannel *types.LineChannel, wg *sync.WaitGroup, ringTimeoutChan chan<- bool) {
	ctx := man.ManagerContext
	lineChannel := ctx.Channel
	cell := ctx.Cell
	log := ctx.Log
	flow:=ctx.Flow
	record := helpers.NewRecording(flow.User,nil)
	_,recordErr:=record.InitiateRecordingForChannel(outboundChannel)

	if recordErr != nil {
		log.Error("error starting recording: " + recordErr.Error())
		return
	}



	log.Debug("Dial source link count: " + strconv.Itoa( len( cell.SourceLinks )))
	log.Debug("Dial target link count: " + strconv.Itoa( len( cell.TargetLinks )))

	answer, _ := utils.FindLinkByName( cell.SourceLinks, "source", "Answer")
	//noAnswer, _ = utils.FindLinkByName( cell.SourceLinks, "source", "No Answer")

	endSub := outboundChannel.Channel.Subscribe(ari.Events.StasisEnd)
	defer endSub.Cancel()
	startSub := outboundChannel.Channel.Subscribe(ari.Events.StasisStart)

	defer startSub.Cancel()
	rootEndSub := lineChannel.Channel.Subscribe(ari.Events.StasisEnd)
	defer rootEndSub.Cancel()

	wg.Done()
	log.Debug("listening for channel events...")

	for {

		select {
			case <-startSub.Events():
				log.Debug("started call..")
				log.Debug("SENDING ANSWER RESP...")
				resp := types.ManagerResponse{
					Channel: outboundChannel,
					Link: answer }
				man.ManagerContext.RecvChannel <- &resp
 				ringTimeoutChan <- true
				 return
			case <-endSub.Events():
				log.Debug("ended call..")
				record.Stop()
				return
			case <-rootEndSub.Events():
				log.Debug("root inded call..")
				return

		}
	}
}
func (man *DialManager) startListeningForRingTimeout(timeout int, outboundChannel *types.LineChannel, wg *sync.WaitGroup, ringTimeoutChan <-chan bool) {
	ctx := man.ManagerContext
	log := ctx.Log
	cell := ctx.Cell
	log.Debug("starting ring timeout checker..")
	log.Debug("timeout set for: " + strconv.Itoa( timeout ))
	noAnswer, _ := utils.FindLinkByName( cell.SourceLinks, "source", "No Answer")

    duration := time.Now().Add(time.Duration( timeout ) * time.Second)

    // Create a context that is both manually cancellable and will signal
    // a cancel at the specified duration.
    ringCtx, cancel := context.WithDeadline(context.Background(), duration)
    defer cancel()
	wg.Done()
	for {
    select {
		case <-ringTimeoutChan:
				log.Debug("bridge in session. stopping ring timeout")
				return
			case <-ringCtx.Done():
				log.Debug("Ring timeout elapsed.. ending all calls")
				resp := types.ManagerResponse{
					Channel: outboundChannel,
					Link: noAnswer }
				man.ManagerContext.RecvChannel <- &resp
				return
    }
}
}

func (man *DialManager) startOutboundCall(callType string) {
	ctx := man.ManagerContext
	cell := ctx.Cell
	model := cell.Model
	log := ctx.Log
	flow := ctx.Flow
	user := flow.User

	log.Debug("startOutboundCall called..")

	callerId := utils.DetermineCallerId( flow.RootCall, model.Data["caller_id"] )
	log.Debug("caller ID was set to: " + callerId)

	valid, err := api.VerifyCallerId(strconv.Itoa( user.Workspace.Id ), callerId)
	if err != nil {
		log.Debug("verify error: " + err.Error())
		return
	}
	if !valid {
		log.Debug("caller id was invalid. user provided: " + callerId)
		return
	}

	numberToCall,err := utils.DetermineNumberToCall( model.Data )
	if err != nil {
		log.Debug("verify error: " + err.Error())
		return
	}
	//key := src.New(ari.ChannelKey, rid.New(rid.Channel))

	log.Debug("Calling: " + numberToCall)

	timeout := utils.ParseRingTimeout( model.Data["timeout"] )

	outChannel := types.LineChannel{}
	outboundChannel, err := ctx.Client.Channel().Create(nil, utils.CreateChannelRequest( numberToCall )	)

	if err != nil {
		log.Debug("error creating outbound channel: " + err.Error())
		return
	}

	domain := user.Workspace.Domain

	var mappedCallType string

	switch ; callType {
	case "Extension":
		mappedCallType = "extension"
	case "Phone Number":
		mappedCallType = "pstn"
		}


	params := types.CallParams{
		From: callerId,
		To: numberToCall,
		Status: "start",
		Direction: "outbound",
		UserId:  flow.User.Id,
		WorkspaceId: flow.User.Workspace.Id,
		ChannelId: outboundChannel.ID() }
	body, err := json.Marshal( params )
	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}

	log.Info("creating outbound call...")
	resp, err := api.SendHttpRequest( "/call/createCall", body )
	outCall, err := utils.CreateCall( resp.Headers.Get("x-call-id"), &outChannel, &params)

	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}



	apiCallId := strconv.Itoa( outCall.CallId )
	headers := utils.CreateSIPHeaders(domain, callerId, mappedCallType, apiCallId)
	outboundChannel, err = outboundChannel.Originate( utils.CreateOriginateRequest(callerId, numberToCall, headers) )

	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}
	outChannel.Channel = outboundChannel
	stopChannel := make( chan bool )
	wg1 := new(sync.WaitGroup)
	wg1.Add(1)
 	go man.manageOutboundCallLeg(&outChannel, wg1, stopChannel)

	wg1.Wait()

	wg2 := new(sync.WaitGroup)
	wg2.Add(1)
	go man.startListeningForRingTimeout(timeout, &outChannel,wg2, stopChannel)
	wg2.Wait()
}


func NewDialManager(mngrCtx *types.Context, flow *types.Flow) (*DialManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := DialManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *DialManager) StartProcessing() {
	log := man.ManagerContext.Log

	cell := man.ManagerContext.Cell
	//flow := man.ManagerContext.Flow
	data := cell.Model.Data
	// create the bridge

	callType := data["call_type"].(types.ModelDataStr)

	log.Debug("processing call type: " + callType.Value)
	log.Debug( "Creating DIAL... ")

	log.Info("channel added to bridge")

	switch ; callType.Value {
	case "Extension":
		man.startOutboundCall(callType.Value) 
	case "Phone Number":
		man.startOutboundCall(callType.Value) 
		}
}

