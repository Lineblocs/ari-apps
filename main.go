package main

import (
	    _ "github.com/joho/godotenv"
		"context"
		"os"
	"sync"
	"fmt"
	"time"
	"strconv"
	"net/http"
	"errors"
	"encoding/json"

	"github.com/rotisserie/eris"

	"github.com/CyCoreSystems/ari/v5"
	 "github.com/CyCoreSystems/ari-proxy/v5/client"
	"github.com/CyCoreSystems/ari/v5/client/native"
	"github.com/CyCoreSystems/ari/v5/rid"
	"lineblocs.com/processor/grpc"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/logger"
	"lineblocs.com/processor/mngrs"
	"lineblocs.com/processor/api"
)

var ariApp = "lineblocs"

var bridge *ari.BridgeHandle

type APIResponse struct {
	Headers http.Header
	Body []byte
}

func logFormattedMsg(msg string) {
 	log := utils.GetLogger()
	log.Debug(fmt.Sprintf("msg = %s", msg))

}


func createARIConnection(connectCtx context.Context) (ari.Client, error) {
	var err error
	var cl ari.Client
	var useProxy bool
 	log := utils.GetLogger()
 	log.Info("Connecting to: " + os.Getenv("ARI_URL"))
	proxy := os.Getenv("ARI_USE_PROXY")
	if proxy != "" {
		useProxy,err = strconv.ParseBool( proxy )
		if err != nil {
			return nil, err
		}
	}
	ctx := context.Background()
	 if useProxy {
		log.Debug("Using ARI proxy!!!");
		natsgw := os.Getenv("NATSGW_URL")
	   cl, err := client.New(ctx,
			client.WithApplication(ariApp),
			client.WithURI(natsgw))
		return cl, err
	}
	log.Info("Directly connecting to ARI server\r\n");
	cl, err = native.Connect(&native.Options{
			Application:  ariApp,
			Username:     os.Getenv("ARI_USERNAME"),
			Password:     os.Getenv("ARI_PASSWORD"),
			URL:          os.Getenv("ARI_URL"),
			WebsocketURL: os.Getenv("ARI_WSURL") })
	return cl,err
 }

func endBridgeCall( lineBridge *types.LineBridge ) {
 	log := utils.GetLogger()
	log.Debug("ending ALL bridge calls..")
	for _, item := range lineBridge.Channels {
		log.Debug("ending call: " + item.Channel.Key().ID)
		utils.SafeHangup( item )
	}

	// TODO:  billing

}

func startListeningForRingTimeout(timeout int, bridge *types.LineBridge, wg *sync.WaitGroup, ringTimeoutChan <-chan bool) {
 	log := utils.GetLogger()
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
					endBridgeCall(bridge)
					return
		}
	}
}

func manageBridge(bridge *types.LineBridge, call *types.Call, lineChannel *types.LineChannel, outboundChannel *types.LineChannel, wg *sync.WaitGroup) {
 	log := utils.GetLogger()
	h := bridge.Bridge

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
	var numChannelsEntered int = 0
	for {
		select {
		case <-destroySub.Events():
			log.Debug("bridge destroyed")
			return
		case e, ok := <-enterSub.Events():
			if !ok {
				log.Error("channel entered subscription closed")
				return
			}

			v := e.(*ari.ChannelEnteredBridge)
			numChannelsEntered += 1

			log.Debug("channel entered bridge", "channel", v.Channel.Name)
			log.Debug("num channels in bridge: " + strconv.Itoa( numChannelsEntered) )

		case e, ok := <-leaveSub.Events():
			if !ok {
				log.Error("channel left subscription closed")
				return
			}
			v := e.(*ari.ChannelLeftBridge)
			log.Debug("channel left bridge", "channel", v.Channel.Name)
			log.Debug("ending all calls in bridge...")
			// end both calls
			utils.SafeHangup( lineChannel )
			utils.SafeHangup( outboundChannel )

			log.Debug("updating call status...")
			api.UpdateCall(call, "ended")
		}
	}
}


func manageOutboundCallLeg(lineChannel *types.LineChannel, outboundChannel *types.LineChannel, lineBridge *types.LineBridge, wg *sync.WaitGroup, ringTimeoutChan chan<- bool) (error) {

 	log := utils.GetLogger()
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
 				ringTimeoutChan <- true
			case <-endSub.Events():
				log.Debug("ended call..")

		}
	}
}


func ensureBridge( cl ari.Client,	src *ari.Key, user *types.User, lineChannel *types.LineChannel, callerId string, numberToCall string, typeOfCall string) (error) {
 	log := utils.GetLogger()
	log.Debug("ensureBridge called..")
	var bridge *ari.BridgeHandle 
	var err error

	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
	bridge, err = cl.Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		bridge = nil
		return eris.Wrap(err, "failed to create bridge")
	}
	outChannel := types.LineChannel{}
	lineBridge := types.NewBridge(bridge)
	
	log.Info("channel added to bridge")
	outboundChannel, err := cl.Channel().Create(nil, utils.CreateChannelRequest( numberToCall )	)

	if err != nil {
		log.Debug("error creating outbound channel: " + err.Error())
		return err
	}


	log.Debug("Originating call...")

	params := types.CallParams{
		From: callerId,
		To: numberToCall,
		Status: "start",
		Direction: "outbound",
		UserId:  user.Id,
		WorkspaceId: user.Workspace.Id,
		ChannelId: outboundChannel.ID() }
	body, err := json.Marshal( params )
	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return err
	}


	log.Info("creating call...")
	log.Info("calling " + numberToCall)
	resp, err := api.SendHttpRequest( "/call/createCall", body)

	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return err
	}
	id := resp.Headers.Get("x-call-id")
	log.Debug("Call ID is: " + id)
	idAsInt, err := strconv.Atoi(id)

	call := types.Call{
		CallId: idAsInt,
		Channel: lineChannel,
		Started: time.Now(),
		Params: &params }

	domain := user.Workspace.Domain
	apiCallId := strconv.Itoa( call.CallId )
	headers := utils.CreateSIPHeaders(domain, callerId, typeOfCall, apiCallId)
	outboundChannel, err = outboundChannel.Originate( utils.CreateOriginateRequest(callerId, numberToCall, headers) )
	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return err
	}

	stopChannel := make( chan bool )
	outChannel.Channel = outboundChannel
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go manageBridge(lineBridge, &call, lineChannel, &outChannel, wg)
	wg.Wait()
	if err := bridge.AddChannel(lineChannel.Channel.Key().ID); err != nil {
		log.Error("failed to add channel to bridge", "error", err)
		return errors.New( "failed to add channel to bridge" )
	}


	log.Info("creating outbound call...")
	resp, err = api.SendHttpRequest( "/call/createCall", body )
	_, err = utils.CreateCall( resp.Headers.Get("x-call-id"), &outChannel, &params)

	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return err
	}



	lineChannel.Channel.Ring()
	wg1 := new(sync.WaitGroup)
	wg1.Add(1)
	utils.AddChannelToBridge( lineBridge, lineChannel )
	utils.AddChannelToBridge( lineBridge, &outChannel )
 	go manageOutboundCallLeg(lineChannel, &outChannel, lineBridge, wg1, stopChannel)
	wg1.Wait()


	timeout := 30
	wg2 := new(sync.WaitGroup)
	wg2.Add(1)
	go startListeningForRingTimeout(timeout, lineBridge, wg2, stopChannel)
	wg2.Wait()

	return nil
}


func main() {
 	log := utils.GetLogger()
	// OPTIONAL: setup logging
	//native.Logger = log

	log.Info("Connecting")
	ctx, cancel := context.WithCancel(context.Background())
	connectCtx, cancel2 := context.WithCancel(context.Background())
	defer cancel()
	defer cancel2()
	cl, err := createARIConnection(connectCtx)
	log.Info("Connected to ARI")

	if err != nil {
		panic( err.Error ());
		return
	}

	defer cl.Close()

	log.Info("starting GRPC listener...")
	go grpc.StartListener(cl)
	// setup app

	log.Info("Starting listener app")

	log.Info("Listening for new calls")
	sub := cl.Bus().Subscribe(nil, "StasisStart")

	for {
		select {
			case e := <-sub.Events():
				v := e.(*ari.StasisStart)
				log.Info("Got stasis start", "channel", v.Channel.ID)
				go startExecution(cl, v, ctx, cl.Channel().Get(v.Key(ari.ChannelKey, v.Channel.ID)))
			case <-ctx.Done():
				return
			case <-connectCtx.Done():
				cl.Close()
				return
		}
	}
}

type bridgeManager struct {
	h *ari.BridgeHandle
}

func createCall() (types.Call, error) {
	return types.Call{}, nil
}
func createCallDebit(user *types.User, call *types.Call, direction string) (error) {
	return nil
}
func attachChannelLifeCycleListeners( flow* types.Flow, channel* types.LineChannel, ctx context.Context, callChannel chan *types.Call) {
 	log := utils.GetLogger()
	var call *types.Call 
	endSub := channel.Channel.Subscribe(ari.Events.StasisEnd)
	defer endSub.Cancel()

	call = nil

	for {

		select {
			case <-ctx.Done():
				return
			case <-endSub.Events():
				log.Debug("stasis end called..")
				call.Ended = time.Now()
				params := types.StatusParams{
					CallId: call.CallId,
					Ip: utils.GetPublicIp(),
					Status: "ended" }
				body, err := json.Marshal( params )
				if err != nil {
					log.Debug("JSON error: " + err.Error())
					continue
				}

				_, err = api.SendHttpRequest( "/call/updateCall", body)
				if err != nil {
					log.Debug("HTTP error: " + err.Error())
					continue
				}
				err = createCallDebit(flow.User, call, "incoming")
				if err != nil {
					log.Debug("HTTP error: " + err.Error())
					continue
				}


			case call = <-callChannel:
				log.Debug("call is setup")
				log.Debug("id is " + strconv.Itoa( call.CallId ))
		}
	}
}
func attachDTMFListeners( channel* types.LineChannel, ctx context.Context) {
 	log := utils.GetLogger()
	dtmfSub := channel.Channel.Subscribe(ari.Events.ChannelDtmfReceived)
	defer dtmfSub.Cancel()

	for {

		select {
			case <-ctx.Done():
				return
			case <-dtmfSub.Events():
				log.Debug("received DTMF!")
		}
	}
}


func processIncomingCall(cl ari.Client, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, exten string, callerId string ) {
 	log := utils.GetLogger()
	go attachDTMFListeners( lineChannel, ctx )
	callChannel := make(chan *types.Call)
	go attachChannelLifeCycleListeners( flow, lineChannel, ctx, callChannel )

	log.Debug("calling API to create call...")
	log.Debug("exten is: " + exten)
	log.Debug("caller ID is: " + callerId)
	params := types.CallParams{
		From: callerId,
		To: exten,
		Status: "start",
		Direction: "inbound",
		UserId:  flow.User.Id,
		WorkspaceId: flow.User.Workspace.Id,
		ChannelId: lineChannel.Channel.ID() }
	body, err := json.Marshal( params )
	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}


	log.Info("creating call...")
	resp, err := api.SendHttpRequest( "/call/createCall", body)
	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}

	id := resp.Headers.Get("x-call-id")
	log.Debug("Call ID is: " + id)
	idAsInt, err := strconv.Atoi(id)

	call := types.Call{
		CallId: idAsInt,
		Channel: lineChannel,
		Started: time.Now(),
		Params: &params }

		flow.RootCall = &call
	log.Debug("answering call..")
	lineChannel.Channel.Answer()
	vars := make( map[string] string )
	go mngrs.ProcessFlow( cl, ctx, flow, lineChannel, vars, flow.Cells[ 0 ])
	callChannel <-  &call
	for {
		select {
			case <-ctx.Done():
				return
		}
	}
}


func startExecution(cl ari.Client, event *ari.StasisStart, ctx context.Context,  h *ari.ChannelHandle) {
 	log := utils.GetLogger()
	log.Info("running app", "channel", h.Key().ID)

	action := event.Args[ 0 ]
	exten := event.Args[ 1 ]
	vals := make(map[string] string)
	vals["number"] = exten

	log.Debug("received action: " + action)
	log.Debug("EXTEN: " + exten)

	switch ; action {
	case "h":
		fmt.Println("Received h handler - not processing")
	case "DID_DIAL":
		fmt.Println("Already dialed - not processing")
		return
	case "DID_DIAL_2":
		fmt.Println("Already dialed - not processing")
	case "INCOMING_CALL":
		body, err := api.SendGetRequest("/user/getDIDNumberData", vals)

		if err != nil {
			log.Error("startExecution err " + err.Error())
			return
		}

		var data types.FlowDIDData
		var flowJson types.FlowVars
		err = json.Unmarshal( []byte(body), &data )
		if err != nil {
			log.Error("startExecution err " + err.Error())
			return
		}

		if utils.CheckFreeTrial( data.Plan ) {
			log.Error("Ending call due to free trial")
			h.Hangup()
			logFormattedMsg(logger.FREE_TRIAL_ENDED)
			return
		}
		err = json.Unmarshal( []byte(data.FlowJson), &flowJson )
		if err != nil {
			log.Error("startExecution err " + err.Error())
			return
		}

		body, err = api.SendGetRequest("/user/getWorkspaceMacros", vals)

		if err != nil {
			log.Error("startExecution err " + err.Error())
			return
		}
		var macros []*types.WorkspaceMacro
		err = json.Unmarshal( []byte(body), &macros)
		if err != nil {
			log.Error("startExecution err " + err.Error())
			return
		}


		lineChannel := types.LineChannel{
			Channel: h }
		user := types.NewUser(data.CreatorId, data.WorkspaceId, data.WorkspaceName)
		flow := types.NewFlow(
			data.FlowId,
			user,
			&flowJson,
			&lineChannel, 
			macros,
			cl)


		log.Debug("processing action: " + action)

		callerId := event.Args[ 2 ]
		fmt.Printf("Starting stasis with extension: %s, caller id: %s", exten, callerId)
		go processIncomingCall( cl, ctx, flow, &lineChannel, exten, callerId )
	case  "OUTGOING_PROXY_ENDPOINT":

		callerId := event.Args[ 2 ]
		domain := event.Args[ 3 ]


		lineChannel := types.LineChannel{
			Channel: h }

			log.Debug("looking up domain: " + domain)
		resp, err := api.GetUserByDomain( domain )

		if err != nil {
			log.Debug("could not get domain. error: " + err.Error())
			return
		}
		log.Debug("workspace ID= " + strconv.Itoa(resp.WorkspaceId))
		user := types.NewUser(resp.Id, resp.WorkspaceId, resp.WorkspaceName)

		fmt.Printf("Received call from %s, domain: %s\r\n", callerId, domain)
		fmt.Printf("Calling %s\r\n", exten)
		h.Answer()
		err = ensureBridge( cl, lineChannel.Channel.Key(), user, &lineChannel, callerId, exten, "extension")
		if err != nil {
			log.Debug("could not create bridge. error: " + err.Error())
			return

		}

	case "OUTGOING_PROXY":
		callerId := event.Args[ 2 ]
		domain := event.Args[ 3 ]

			log.Debug("channel key: " + h.Key().ID)

		lineChannel := types.LineChannel{
			Channel: h }
		resp, err := api.GetUserByDomain( domain )

		if err != nil {
			log.Debug("could not get domain. error: " + err.Error())
			return
		}
		log.Debug("workspace ID= " + strconv.Itoa(resp.WorkspaceId))
		user := types.NewUser(resp.Id, resp.WorkspaceId, resp.WorkspaceName)

		fmt.Printf("Received call from %s, domain: %s\r\n", callerId, domain)

		callerInfo, err := api.GetCallerId(user.Workspace.Domain, callerId)

		if err != nil {
			log.Debug("could not get caller id. error: " + err.Error())
			return
		}
		fmt.Printf("setup caller id: " + callerInfo.CallerId)
		h.Answer()
			err=ensureBridge( cl, lineChannel.Channel.Key(), user, &lineChannel, callerInfo.CallerId, exten, "pstn")
if err != nil {
			log.Debug("could not create bridge. error: " + err.Error())
			return

		}

	case "OUTGOING_PROXY_MEDIA":
		log.Info("media service call..")

	default:
		log.Info("unknown call type...")
	}
}