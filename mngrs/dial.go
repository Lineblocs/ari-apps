package mngrs

import (
	//"context"

	"encoding/json"
	"strconv"
	"sync"

	"github.com/CyCoreSystems/ari/v5"
	helpers "github.com/Lineblocs/go-helpers"
	"github.com/sirupsen/logrus"
	"lineblocs.com/processor/api"
	"lineblocs.com/processor/resources"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)

type DialManager struct {
	ManagerContext *types.Context
	Flow           *types.Flow
}

func (man *DialManager) manageOutboundCallLeg(outboundChannel *types.LineChannel, outCall *types.Call, wg *sync.WaitGroup, ringTimeoutChan chan<- bool) {
	ctx := man.ManagerContext
	lineChannel := ctx.Channel
	cell := ctx.Cell
	flow := ctx.Flow
	storageServer := types.StorageServer{
		Ip: utils.GetARIHost(),
	}
	producer := ctx.EventProducer
	record := resources.NewRecording(&storageServer, producer, flow.User, &outCall.CallId, false)
	_, recordErr := record.InitiateRecordingForChannel(outboundChannel)

	if recordErr != nil {
		helpers.Log(logrus.ErrorLevel, "error starting recording: "+recordErr.Error())
		return
	}

	helpers.Log(logrus.DebugLevel, "Dial source link count: "+strconv.Itoa(len(cell.SourceLinks)))
	helpers.Log(logrus.DebugLevel, "Dial target link count: "+strconv.Itoa(len(cell.TargetLinks)))

	answer, _ := utils.FindLinkByName(cell.SourceLinks, "source", "Answer")
	//noAnswer, _ = utils.FindLinkByName( cell.SourceLinks, "source", "No Answer")

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
			helpers.Log(logrus.DebugLevel, "SENDING ANSWER RESP...")
			resp := types.ManagerResponse{
				Channel: outboundChannel,
				Link:    answer}
			man.ManagerContext.RecvChannel <- &resp
			ringTimeoutChan <- true
			return
		case <-endSub.Events():
			helpers.Log(logrus.DebugLevel, "ended call..")
			record.Stop()
			return
		case <-rootEndSub.Events():
			helpers.Log(logrus.DebugLevel, "root inded call..")
			return

		}
	}
}

func (man *DialManager) startOutboundCall(callType string) {
	ctx := man.ManagerContext
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

	outChannel := types.NewChannel(nil, true)
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
	wg1 := new(sync.WaitGroup)
	wg1.Add(1)
	go man.manageOutboundCallLeg(&outChannel, outCall, wg1, stopChannel)

	wg1.Wait()

	wg2 := new(sync.WaitGroup)
	wg2.Add(1)
	noAnswer, _ := utils.FindLinkByName(cell.SourceLinks, "source", "No Answer")
	go outChannel.StartRingTimer(ctx, noAnswer, timeout, wg2, stopChannel, "dial")
	wg2.Wait()
}

func NewDialManager(mngrCtx *types.Context, flow *types.Flow) *DialManager {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := DialManager{
		ManagerContext: mngrCtx,
		Flow:           flow}
	return &item
}
func (man *DialManager) StartProcessing() {

	cell := man.ManagerContext.Cell
	//flow := man.ManagerContext.Flow
	data := cell.Model.Data
	// create the bridge

	callType := data["call_type"].(types.ModelDataStr)

	helpers.Log(logrus.DebugLevel, "processing call type: "+callType.Value)
	helpers.Log(logrus.DebugLevel, "Creating DIAL... ")

	helpers.Log(logrus.InfoLevel, "channel added to bridge")

	switch callType.Value {
	case "Extension":
		man.startOutboundCall(callType.Value)
	case "Phone Number":
		man.startOutboundCall(callType.Value)
	}
}
