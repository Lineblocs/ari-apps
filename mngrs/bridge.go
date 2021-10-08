package mngrs
import (
	//"context"
	"sync"
	"errors"
	"strconv"
	"encoding/json"
	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/rid"
	"github.com/CyCoreSystems/ari/v5/ext/play"
	"github.com/rotisserie/eris"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/api"
)
type BridgeManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func (man *BridgeManager) ensureBridge(src *ari.Key) (error) {
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
	go man.manageBridge(&lineBridge, wg)
	wg.Wait()
	if err := bridge.AddChannel(ctx.Channel.Channel.Key().ID); err != nil {
		log.Error("failed to add channel to bridge", "error", err)
		return errors.New( "failed to add channel to bridge" )
	}

	log.Info("channel added to bridge")


	return nil
}
func (man *BridgeManager) manageBridge(bridge *types.LineBridge, wg *sync.WaitGroup) {
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
			go man.startOutboundCall(bridge, wg) 
			
			func() {
				log.Debug("Playing sound...")
				if err := play.Play(ctx.Context, h, play.URI("sound:hello-world")).Err(); err != nil {
					log.Error("failed to play join sound", "error", err)
				}
			}()
		case e, ok := <-leaveSub.Events():
			if !ok {
				log.Error("channel left subscription closed")
				return
			}
			v := e.(*ari.ChannelLeftBridge)
			log.Debug("channel left bridge", "channel", v.Channel.Name)
			go func() {
				if err := play.Play(ctx.Context, h, play.URI("sound:confbridge-leave")).Err(); err != nil {
					log.Error("failed to play leave sound", "error", err)
				}
			}()
		}
	}
}

func (man *BridgeManager) startOutboundCall(bridge *types.LineBridge, wg *sync.WaitGroup) {
	ctx := man.ManagerContext
	cell := ctx.Cell
	model := cell.Model
	log := ctx.Log
	flow := ctx.Flow
	user := flow.User
	log.Debug("startOutboundCall called..")

	callerId := utils.DetermineCallerId( flow.RootCall, model.Data["caller_id"].ValueStr )

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
	outChannel := types.LineChannel{
		Channel: outboundChannel }
	_, err = utils.CreateCall( resp.Headers.Get("x-call-id"), &outChannel, &params)

	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}

	outboundChannel.Originate( utils.CreateOriginateRequest(callerId, numberToCall) )
	return

	/*
	startSub := outChannel.Channel.Subscribe(ari.Events.StasisStart)
	defer startSub.Cancel()
	destSub := outChannel.Channel.Subscribe(ari.Events.ChannelDestroyed)
	defer destSub.Cancel()
	endSub := outChannel.Channel.Subscribe(ari.Events.StasisEnd)
	defer endSub.Cancel()

	for {

		select {
			case <-startSub.Events():
				log.Debug("call is setup")
			case <-endSub.Events():
				log.Debug("call ended..")
			case <-destSub.Events():
				log.Debug("call destroyed..")
		}

	}
	*/



	if err := bridge.Bridge.AddChannel(outChannel.Channel.Key().ID); err != nil {
		log.Error("failed to add channel to bridge", "error", err)
		return
	}
	log.Debug("added outbound channel to bridge..")
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
	channel := man.ManagerContext.Channel
	data := cell.Model.Data
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Connected Call Ended")
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Caller Hung Up")
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Declined")

	// create the bridge

	callType := data["call_type"]
	_ = types.NewRecording(flow.User, channel, true)

	log.Debug("processing call type: " + callType.ValueStr)
	if callType.ValueStr == "Extension" || callType.ValueStr == "Phone Number" || callType.ValueStr == "Extension Flow"  {
		man.startSimpleCall()
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
func (man *BridgeManager) startSimpleCall() {
	log := man.ManagerContext.Log
	log.Debug("Starting simple call..")
	man.ensureBridge(man.ManagerContext.Channel.Channel.Key())
}