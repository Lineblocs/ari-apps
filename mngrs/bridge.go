package mngrs
import (
	//"context"
	"sync"
	"errors"
	"strconv"
	"context"
	"time"
	"strings"
	"encoding/json"
	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/rid"
	//"github.com/CyCoreSystems/ari/v5/ext/play"
	"github.com/rotisserie/eris"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/api"
	"lineblocs.com/processor/helpers"
)
type BridgeManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func (man *BridgeManager) ensureBridge(src *ari.Key, callType string) (error) {
	ctx := man.ManagerContext
	log := ctx.Log

	log.Debug("ensureBridge called..")
	var bridge *ari.BridgeHandle 
	var err error

	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
	bridge, err = ctx.Client.Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		bridge = nil
		return eris.Wrap(err, "failed to create bridge")
	}

	lineBridge := types.NewBridge(bridge)
	log.Info("channel added to bridge")

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go man.manageBridge(lineBridge, wg, callType)
	wg.Wait()
	if err := bridge.AddChannel(ctx.Channel.Channel.Key().ID); err != nil {
		log.Error("failed to add channel to bridge", "error", err)
		return errors.New( "failed to add channel to bridge" )
	}

	log.Info("channel added to bridge")
	man.addAllRequestedCalls(lineBridge);
	go man.startOutboundCall(lineBridge, callType) 



	return nil
}

func (man *BridgeManager) addAllRequestedCalls(bridge *types.LineBridge) {
	ctx := man.ManagerContext
	log := ctx.Log
	cell := ctx.Cell
	data := cell.Model.Data
	extras := data["extra_call_ids"]
	if extras == nil {
		return
	}
	callIds := extras.(types.ModelDataStr)
	log.Debug("looking up requested channels")
	if callIds.Value != "" {
		ids := strings.Split(callIds.Value, ",")
		for _, id := range ids {
			log.Debug("adding requested channel: " + id)
			call, err := api.FetchCall( id )
			if err != nil {
				log.Debug("error fetching requested channel: " + err.Error())
				continue
			}

			key := ari.NewKey(ari.ChannelKey, call.ChannelId)
			channel := ctx.Client.Channel().Get(key)
			reqChannel := types.LineChannel{ Channel: channel }
			utils.AddChannelToBridge( bridge, &reqChannel )
		}
	}
}
func (man *BridgeManager) manageBridge(bridge *types.LineBridge, wg *sync.WaitGroup, callType string) {
	h := bridge.Bridge
	ctx := man.ManagerContext
	flow:=ctx.Flow
	cell := ctx.Cell
	channel := ctx.Channel
	log := ctx.Log
	record := helpers.NewRecording(flow.User,&flow.RootCall.CallId)
	//_,recordErr:=record.InitiateRecordingForBridge(bridge)
	_,recordErr:=record.InitiateRecordingForBridge(bridge)
	next, _ := utils.FindLinkByName( cell.TargetLinks, "source", "Connected Call Ended")

	if recordErr != nil {
		log.Error("error starting recording: " + recordErr.Error())
		return
	}

	log.Debug("manageBridge called..")
	// Delete the bridge when we exit
	defer h.Delete()

	destroySub := h.Subscribe(ari.Events.BridgeDestroyed)
	defer destroySub.Cancel()

	enterSub := h.Subscribe(ari.Events.ChannelEnteredBridge)
	defer enterSub.Cancel()

	leaveSub := h.Subscribe(ari.Events.ChannelLeftBridge)
	defer leaveSub.Cancel()


	wg.Done()
	log.Debug("listening for bridge events...")
	for {
		select {
		case <-ctx.Context.Done():
			return
		case <-destroySub.Events():
			log.Debug("bridge destroyed")
			return
		case e, ok := <-enterSub.Events():
			if !ok {
				log.Error("channel entered subscription closed")
				return
			}
			v := e.(*ari.ChannelEnteredBridge)
			log.Debug("channel entered bridge", "channel", v.Channel.Name)
		case e, ok := <-leaveSub.Events():
			if !ok {
				log.Error("channel left subscription closed")
				return
			}
			v := e.(*ari.ChannelLeftBridge)
			log.Debug("channel left bridge", "channel", v.Channel.Name)
			man.endBridgeCall(bridge)
			record.Stop()

			resp := types.ManagerResponse{
				Channel: channel,
				Link: next }
			man.ManagerContext.RecvChannel <- &resp
		}
	}
}

func (man *BridgeManager) endBridgeCall( lineBridge *types.LineBridge ) {
	ctx := man.ManagerContext
	log := ctx.Log
	log.Debug("ending ALL bridge calls..")
	for _, item := range lineBridge.Channels {
		log.Debug("ending call: " + item.Channel.Key().ID)
		utils.SafeHangup( item )
	}

	// TODO:  billing

}
func (man *BridgeManager) manageOutboundCallLeg(outboundChannel *types.LineChannel, lineBridge *types.LineBridge, wg *sync.WaitGroup, ringTimeoutChan chan<- bool) {
	ctx := man.ManagerContext
	lineChannel := ctx.Channel
	cell := ctx.Cell
	log := ctx.Log

	next, err := utils.FindLinkByName( cell.TargetLinks, "source", "Caller Hung Up")
	if err != nil {
		log.Debug("error finding link.. ")
	}

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

				if err := lineBridge.Bridge.AddChannel(outboundChannel.Channel.Key().ID); err != nil {
					log.Error("failed to add channel to bridge", "error", err)
					return
				}
				log.Debug("added outbound channel to bridge..")
				log.Debug("exiting...")
				lineChannel.Channel.StopRing()
 				ringTimeoutChan <- true
				 return
			case <-endSub.Events():
				log.Debug("ended call..")
				return
			case <-rootEndSub.Events():
				log.Debug("root inded call..")
			resp := types.ManagerResponse{
				Channel: lineChannel,
				Link: next }
				man.ManagerContext.RecvChannel <- &resp
				return

		}
	}
}

func (man *BridgeManager) startOutboundCall(bridge *types.LineBridge,callType string) {
	ctx := man.ManagerContext
	channel := ctx.Channel
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

	numberToCall, err := utils.DetermineNumberToCall( model.Data )
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

	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}
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
	channel.Channel.Ring()

	wg1 := new(sync.WaitGroup)
	wg1.Add(1)
	utils.AddChannelToBridge( bridge, channel )
	utils.AddChannelToBridge( bridge, &outChannel )
 	go man.manageOutboundCallLeg(&outChannel, bridge, wg1, stopChannel)

	wg1.Wait()

	wg2 := new(sync.WaitGroup)
	wg2.Add(1)
	go man.startListeningForRingTimeout(timeout, bridge, wg2, stopChannel)
	wg2.Wait()
}

func (man *BridgeManager) startListeningForRingTimeout(timeout int, bridge *types.LineBridge, wg *sync.WaitGroup, ringTimeoutChan <-chan bool) {
	ctx := man.ManagerContext
	log := ctx.Log
	log.Debug("starting ring timeout checker..")
	log.Debug("timeout set for: " + strconv.Itoa( timeout ))
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
				man.endBridgeCall(bridge)
				return
    }
}
}

func NewBridgeManager(mngrCtx *types.Context, flow *types.Flow) (*BridgeManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := BridgeManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *BridgeManager) StartProcessing() {
	log := man.ManagerContext.Log
	log.Debug( "Creating bridge... ")
	cell := man.ManagerContext.Cell
	flow := man.ManagerContext.Flow
	user := flow.User
	data := cell.Model.Data
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Connected Call Ended")
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Caller Hung Up")
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Declined")

	// create the bridge

	callType := data["call_type"].(types.ModelDataStr)

	log.Debug("processing call type: " + callType.Value)
	if callType.Value == "Extension" || callType.Value == "Phone Number" {
		man.startSimpleCall(callType.Value)

	} else if callType.Value == "ExtensionFlow" {
		extension := data["extension"].(types.ModelDataStr).Value
		man.initiateExtFlow(user, extension)
	} else if callType.Value == "Follow Me" {
	} else if callType.Value == "Queue" {
	} else if callType.Value == "Merge Calls" {
		man.startCallMerge(callType.Value)
	}


	for {
		select {
			case <-man.ManagerContext.Context.Done():
				return
		}
	}
}
func (man *BridgeManager) startSimpleCall(callType string) {
	log := man.ManagerContext.Log
	log.Debug("Starting simple call..")
	man.ensureBridge(man.ManagerContext.Channel.Channel.Key(), callType)
}

func (man *BridgeManager) initiateExtFlow(user *types.User, extension string) {
	ctx := man.ManagerContext
	log := man.ManagerContext.Log
	log.Debug("Starting new ext flow..")
	workspace := user.Workspace.Id
	channel := ctx.Channel
	client := ctx.Client
	coreFlow := ctx.Flow

	subFlow, err := api.GetExtensionFlowInfo(strconv.Itoa(workspace), extension)
	if err != nil {
		log.Debug("error starting new flow: " + err.Error())
		return
	}
	info := subFlow.Vars
	//user := types.NewUser(data.CreatorId, data.WorkspaceId, data.WorkspaceName)
	flow := types.NewFlow(
		subFlow.FlowId,	
		user,
		info,
		channel, 
		coreFlow.WorkspaceFns,
		client)

	vars := make( map[string] string )
	go ProcessFlow( client, man.ManagerContext.Context, flow, channel, vars, flow.Cells[ 0 ])
}

func (man *BridgeManager) startCallMerge(callType string) {
	ctx := man.ManagerContext
	log := man.ManagerContext.Log
	log.Debug("Starting call merge")

	flow := ctx.Flow
	id := flow.FlowId
	cell := ctx.Cell
	bridgeKey := "bridge-" + cell.Cell.Name + "-" + strconv.Itoa( id ) 
	key := ari.NewKey(ari.BridgeKey, bridgeKey)
	bridge, err := ctx.Client.Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		bridge = nil
		log.Debug("failed to create bridge")
		return
	}

	lineBridge := types.NewBridge(bridge)
	log.Info("channel added to bridge")

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go man.manageBridge(lineBridge, wg, callType)
	wg.Wait()


	man.addAllRequestedCalls(lineBridge);
}