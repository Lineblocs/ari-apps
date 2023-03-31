package mngrs

import (
	//"context"

	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/rid"
	"github.com/sirupsen/logrus"

	//"github.com/CyCoreSystems/ari/v5/ext/play"
	helpers "github.com/Lineblocs/go-helpers"
	"github.com/rotisserie/eris"
	"lineblocs.com/processor/api"
	processor_helpers "lineblocs.com/processor/helpers"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)

type BridgeManager struct {
	ManagerContext *types.Context
	Flow           *types.Flow
}

func (man *BridgeManager) ensureBridge(src *ari.Key, callType string) error {
	ctx := man.ManagerContext
	helpers.Log(logrus.DebugLevel, "ensureBridge called..")
	var bridge *ari.BridgeHandle
	var err error

	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
	bridge, err = ctx.Client.Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		bridge = nil
		return eris.Wrap(err, "failed to create bridge")
	}

	lineBridge := types.NewBridge(bridge)
	helpers.Log(logrus.InfoLevel, "channel added to bridge")

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go man.manageBridge(lineBridge, wg, callType)
	wg.Wait()
	if err := bridge.AddChannel(ctx.Channel.Channel.Key().ID); err != nil {
		helpers.Log(logrus.ErrorLevel, "failed to add channel to bridge, error:"+err.Error())
		return errors.New("failed to add channel to bridge")
	}

	helpers.Log(logrus.InfoLevel, "channel added to bridge")
	man.addAllRequestedCalls(lineBridge)
	go man.startOutboundCall(lineBridge, callType)

	return nil
}

func (man *BridgeManager) addAllRequestedCalls(bridge *types.LineBridge) {
	ctx := man.ManagerContext
	cell := ctx.Cell
	data := cell.Model.Data
	extras := data["extra_call_ids"]
	if extras == nil {
		return
	}
	callIds := extras.(types.ModelDataStr)
	helpers.Log(logrus.DebugLevel, "looking up requested channels")
	if callIds.Value != "" {
		ids := strings.Split(callIds.Value, ",")
		for _, id := range ids {
			helpers.Log(logrus.DebugLevel, "adding requested channel: "+id)
			call, err := api.FetchCall(id)
			if err != nil {
				helpers.Log(logrus.DebugLevel, "error fetching requested channel: "+err.Error())
				continue
			}

			key := ari.NewKey(ari.ChannelKey, call.ChannelId)
			channel := ctx.Client.Channel().Get(key)
			reqChannel := types.LineChannel{Channel: channel}
			bridge.AddChannel(&reqChannel)
		}
	}
}
func (man *BridgeManager) manageBridge(bridge *types.LineBridge, wg *sync.WaitGroup, callType string) {
	h := bridge.Bridge
	ctx := man.ManagerContext
	flow := ctx.Flow
	cell := ctx.Cell
	channel := ctx.Channel
	record := processor_helpers.NewRecording(flow.User, &flow.RootCall.CallId, false)
	//_,recordErr:=record.InitiateRecordingForBridge(bridge)
	_, recordErr := record.InitiateRecordingForBridge(bridge)
	next, _ := utils.FindLinkByName(cell.TargetLinks, "source", "Connected Call Ended")

	if recordErr != nil {
		helpers.Log(logrus.ErrorLevel, "error starting recording: "+recordErr.Error())
		return
	}

	helpers.Log(logrus.DebugLevel, "manageBridge called..")
	// Delete the bridge when we exit
	defer h.Delete()

	destroySub := h.Subscribe(ari.Events.BridgeDestroyed)
	defer destroySub.Cancel()

	enterSub := h.Subscribe(ari.Events.ChannelEnteredBridge)
	defer enterSub.Cancel()

	leaveSub := h.Subscribe(ari.Events.ChannelLeftBridge)
	defer leaveSub.Cancel()

	wg.Done()
	helpers.Log(logrus.DebugLevel, "listening for bridge events...")
	for {
		select {
		case <-ctx.Context.Done():
			return
		case <-destroySub.Events():
			helpers.Log(logrus.DebugLevel, "bridge destroyed")
			return
		case e, ok := <-enterSub.Events():
			if !ok {
				helpers.Log(logrus.ErrorLevel, "channel entered subscription closed")
				return
			}
			v := e.(*ari.ChannelEnteredBridge)
			helpers.Log(logrus.DebugLevel, "channel entered bridge, channel:"+v.Channel.Name)
		case e, ok := <-leaveSub.Events():
			if !ok {
				helpers.Log(logrus.ErrorLevel, "channel left subscription closed")
				return
			}
			v := e.(*ari.ChannelLeftBridge)
			helpers.Log(logrus.DebugLevel, "channel left bridge, channel:"+v.Channel.Name)
			bridge.EndBridgeCall()
			record.Stop()

			resp := types.ManagerResponse{
				Channel: channel,
				Link:    next}
			man.ManagerContext.RecvChannel <- &resp
		}
	}
}

func (man *BridgeManager) manageOutboundCallLeg(outboundChannel *types.LineChannel, lineBridge *types.LineBridge, wg *sync.WaitGroup, ringTimeoutChan chan<- bool) {
	ctx := man.ManagerContext
	lineChannel := ctx.Channel
	cell := ctx.Cell

	next, err := utils.FindLinkByName(cell.TargetLinks, "source", "Caller Hung Up")
	if err != nil {
		helpers.Log(logrus.DebugLevel, "error finding link.. ")
	}

	endSub := outboundChannel.Channel.Subscribe(ari.Events.StasisEnd)
	defer endSub.Cancel()
	startSub := outboundChannel.Channel.Subscribe(ari.Events.StasisStart)

	defer startSub.Cancel()
	rootEndSub := lineChannel.Channel.Subscribe(ari.Events.StasisEnd)
	defer rootEndSub.Cancel()

	wg.Done()
	helpers.Log(logrus.DebugLevel, "listening for channel events...")

	for {

		select {
		case <-startSub.Events():
			helpers.Log(logrus.DebugLevel, "started call..")

			if err := lineBridge.Bridge.AddChannel(outboundChannel.Channel.Key().ID); err != nil {
				helpers.Log(logrus.ErrorLevel, "failed to add channel to bridge, error:"+err.Error())
				return
			}
			helpers.Log(logrus.DebugLevel, "added outbound channel to bridge..")
			helpers.Log(logrus.DebugLevel, "exiting...")
			lineChannel.Channel.StopRing()
			ringTimeoutChan <- true
			return
		case <-endSub.Events():
			helpers.Log(logrus.DebugLevel, "ended call..")
			return
		case <-rootEndSub.Events():
			helpers.Log(logrus.DebugLevel, "root inded call..")
			resp := types.ManagerResponse{
				Channel: lineChannel,
				Link:    next}
			man.ManagerContext.RecvChannel <- &resp
			return

		}
	}
}

func (man *BridgeManager) startOutboundCall(bridge *types.LineBridge, callType string) {
	ctx := man.ManagerContext
	channel := ctx.Channel
	cell := ctx.Cell
	model := cell.Model
	flow := ctx.Flow
	user := flow.User
	helpers.Log(logrus.DebugLevel, "startOutboundCall called..")
	callerId := utils.DetermineCallerId(flow.RootCall, model.Data["caller_id"])
	helpers.Log(logrus.DebugLevel, "caller ID was set to: "+callerId)

	valid, err := api.VerifyCallerId(strconv.Itoa(user.Workspace.Id), callerId)
	if err != nil {
		helpers.Log(logrus.DebugLevel, "verify error: "+err.Error())
		return
	}
	if !valid {
		helpers.Log(logrus.DebugLevel, "caller id was invalid. user provided: "+callerId)
		return
	}

	numberToCall, err := utils.DetermineNumberToCall(model.Data)
	if err != nil {
		helpers.Log(logrus.DebugLevel, "verify error: "+err.Error())
		return
	}
	//key := src.New(ari.ChannelKey, rid.New(rid.Channel))

	helpers.Log(logrus.DebugLevel, "Calling: "+numberToCall)

	timeout := utils.ParseRingTimeout(model.Data["timeout"])
	outChannel := types.LineChannel{}
	outboundChannel, err := ctx.Client.Channel().Create(nil, utils.CreateChannelRequest(numberToCall))

	if err != nil {
		helpers.Log(logrus.DebugLevel, "error creating outbound channel: "+err.Error())
		return
	}

	domain := user.Workspace.Domain

	var mappedCallType string
	switch callType {
	case "Extension":
		mappedCallType = "extension"
	case "Phone Number":
		mappedCallType = "pstn"
	}

	params := types.CallParams{
		From:        callerId,
		To:          numberToCall,
		Status:      "start",
		Direction:   "outbound",
		UserId:      flow.User.Id,
		WorkspaceId: flow.User.Workspace.Id,
		ChannelId:   outboundChannel.ID()}
	body, err := json.Marshal(params)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return
	}

	helpers.Log(logrus.InfoLevel, "creating outbound call...")
	resp, err := api.SendHttpRequest("/call/createCall", body)

	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return
	}
	outCall, err := outChannel.CreateCall(resp.Headers.Get("x-call-id"), &params)

	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return
	}

	apiCallId := strconv.Itoa(outCall.CallId)
	headers := utils.CreateSIPHeaders(domain, callerId, mappedCallType, apiCallId, nil)
	outboundChannel, err = outboundChannel.Originate(utils.CreateOriginateRequest(callerId, numberToCall, headers))

	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return
	}
	outChannel.Channel = outboundChannel

	stopChannel := make(chan bool)
	channel.Channel.Ring()

	wg1 := new(sync.WaitGroup)
	wg1.Add(1)
	bridge.AddChannel(channel)
	bridge.AddChannel(&outChannel)
	go man.manageOutboundCallLeg(&outChannel, bridge, wg1, stopChannel)

	wg1.Wait()

	wg2 := new(sync.WaitGroup)
	wg2.Add(1)
	go bridge.StartWaitingForRingTimeout(timeout, wg2, stopChannel)
	wg2.Wait()
}

func NewBridgeManager(mngrCtx *types.Context, flow *types.Flow) *BridgeManager {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := BridgeManager{
		ManagerContext: mngrCtx,
		Flow:           flow}
	return &item
}
func (man *BridgeManager) StartProcessing() {
	helpers.Log(logrus.DebugLevel, "Creating bridge... ")
	cell := man.ManagerContext.Cell
	flow := man.ManagerContext.Flow
	user := flow.User
	data := cell.Model.Data
	_, _ = utils.FindLinkByName(cell.SourceLinks, "source", "Connected Call Ended")
	_, _ = utils.FindLinkByName(cell.SourceLinks, "source", "Caller Hung Up")
	_, _ = utils.FindLinkByName(cell.SourceLinks, "source", "Declined")

	// create the bridge

	callType := data["call_type"].(types.ModelDataStr)

	helpers.Log(logrus.DebugLevel, "processing call type: "+callType.Value)
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
	helpers.Log(logrus.DebugLevel, "Starting simple call..")
	man.ensureBridge(man.ManagerContext.Channel.Channel.Key(), callType)
}

func (man *BridgeManager) initiateExtFlow(user *types.User, extension string) {
	ctx := man.ManagerContext
	helpers.Log(logrus.DebugLevel, "Starting new ext flow..")
	workspace := user.Workspace.Id
	channel := ctx.Channel
	client := ctx.Client
	coreFlow := ctx.Flow

	subFlow, err := api.GetExtensionFlowInfo(strconv.Itoa(workspace), extension)
	if err != nil {
		helpers.Log(logrus.DebugLevel, "error starting new flow: "+err.Error())
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

	vars := make(map[string]string)
	go ProcessFlow(client, man.ManagerContext.Context, flow, channel, vars, flow.Cells[0])
}

func (man *BridgeManager) startCallMerge(callType string) {
	ctx := man.ManagerContext
	helpers.Log(logrus.DebugLevel, "Starting call merge")

	flow := ctx.Flow
	id := flow.FlowId
	cell := ctx.Cell
	bridgeKey := "bridge-" + cell.Cell.Name + "-" + strconv.Itoa(id)
	key := ari.NewKey(ari.BridgeKey, bridgeKey)
	bridge, err := ctx.Client.Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		bridge = nil
		helpers.Log(logrus.DebugLevel, "failed to create bridge")
		return
	}

	lineBridge := types.NewBridge(bridge)
	helpers.Log(logrus.InfoLevel, "channel added to bridge")

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go man.manageBridge(lineBridge, wg, callType)
	wg.Wait()

	man.addAllRequestedCalls(lineBridge)
}
