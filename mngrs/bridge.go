package mngrs
import (
	//"context"
	"sync"
	"errors"
	"strconv"
	"encoding/json"
	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/rid"
	//"github.com/CyCoreSystems/ari/v5/ext/play"
	"github.com/rotisserie/eris"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/api"
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

	lineBridge := types.LineBridge{Bridge: bridge}
	log.Info("channel added to bridge")

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go man.manageBridge(&lineBridge, wg, callType)
	wg.Wait()
	if err := bridge.AddChannel(ctx.Channel.Channel.Key().ID); err != nil {
		log.Error("failed to add channel to bridge", "error", err)
		return errors.New( "failed to add channel to bridge" )
	}

	log.Info("channel added to bridge")
	go man.startOutboundCall(&lineBridge, callType) 



	return nil
}
func (man *BridgeManager) manageBridge(bridge *types.LineBridge, wg *sync.WaitGroup, callType string) {
	h := bridge.Bridge
	ctx := man.ManagerContext
	log := ctx.Log


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
		}
	}
}

func (man *BridgeManager) manageOutboundCallLeg(outboundChannel *types.LineChannel, lineBridge *types.LineBridge, wg *sync.WaitGroup) (error) {
	ctx := man.ManagerContext
	lineChannel := ctx.Channel
	log := ctx.Log
	endSub := outboundChannel.Channel.Subscribe(ari.Events.StasisEnd)
	defer endSub.Cancel()
	startSub := outboundChannel.Channel.Subscribe(ari.Events.StasisStart)

	defer startSub.Cancel()
	wg.Done()
	log.Debug("listening for channel events...")

	for {

		select {
			case <-startSub.Events():
				log.Debug("started call..")

				if err := lineBridge.Bridge.AddChannel(outboundChannel.Channel.Key().ID); err != nil {
					log.Error("failed to add channel to bridge", "error", err)
					return err
				}
				log.Debug("added outbound channel to bridge..")
				lineChannel.Channel.StopRing()
			case <-endSub.Events():
				log.Debug("ended call..")

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

	callerId := utils.DetermineCallerId( flow.RootCall, model.Data["caller_id"].ValueStr )
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

	numberToCall := utils.DetermineNumberToCall( model.Data )
	//key := src.New(ari.ChannelKey, rid.New(rid.Channel))

	log.Debug("Calling: " + numberToCall)


	/*
	src := channel.Channel.Key()

	key := src.New(ari.ChannelKey, rid.New(rid.Channel))
	outboundChannel := ari.NewChannelHandle( key, ctx.Client.Channel(), nil )
	*/
	outboundChannel, err := ctx.Client.Channel().Create(nil, utils.CreateChannelRequest( numberToCall )	)

	if err != nil {
		log.Debug("error creating outbound channel: " + err.Error())
		return
	}

	params := types.CallParams{
		From: callerId,
		To: numberToCall,
		Status: "start",
		Direction: "outbound",
		UserId:  flow.User.Id,
		WorkspaceId: flow.User.Workspace.Id }
	body, err := json.Marshal( params )
	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}

	log.Info("creating outbound call...")
	resp, err := api.SendHttpRequest( "/call/createCall", body )
	outChannel := types.LineChannel{}
	_, err = utils.CreateCall( resp.Headers.Get("x-call-id"), &outChannel, &params)

	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}

	domain := user.Workspace.Domain

	var mappedCallType string
	if callType == "Extension" {
		mappedCallType = "extension"
	} else if callType == "Phone Number" {
		mappedCallType = "pstn"
	}
	headers := utils.CreateSIPHeaders(domain, callerId, mappedCallType)
	outboundChannel, err = outboundChannel.Originate( utils.CreateOriginateRequest(callerId, numberToCall, headers) )

	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}
	outChannel.Channel = outboundChannel

	channel.Channel.Ring()
	wg2 := new(sync.WaitGroup)
	wg2.Add(1)
 	go man.manageOutboundCallLeg(&outChannel, bridge, wg2)
	wg2.Wait()
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
	channel := man.ManagerContext.Channel
	data := cell.Model.Data
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Connected Call Ended")
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Caller Hung Up")
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Declined")

	// create the bridge

	callType := data["call_type"]
	_ = types.NewRecording(flow.User, channel, true)

	log.Debug("processing call type: " + callType.ValueStr)
	if callType.ValueStr == "Extension" || callType.ValueStr == "Phone Number" {
		man.startSimpleCall(callType.ValueStr)

	} else if callType.ValueStr == "ExtensionFlow" {
		extension := data["extension"].ValueStr
		man.initiateExtFlow(user, extension)
	} else if callType.ValueStr == "Follow Me" {
	} else if callType.ValueStr == "Queue" {
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

	info, err := api.GetExtensionFlowInfo(strconv.Itoa(workspace), extension)
	if err != nil {
		log.Debug("error starting new flow: " + err.Error())
		return
	}
	//user := types.NewUser(data.CreatorId, data.WorkspaceId, data.WorkspaceName)
	flow := types.NewFlow(
		user,
		info,
		channel, 
		client)

	vars := make( map[string] string )
	go ProcessFlow( client, man.ManagerContext.Context, flow, channel, vars, flow.Cells[ 0 ])
	/*
	for {
		select {
			case <-ctx.Done():
				return
		}
	}
	*/

}