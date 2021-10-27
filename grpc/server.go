package grpc;

import (
	"fmt"
	"time"
	"errors"
	"strconv"
	"github.com/google/uuid"
	"encoding/json"
	"golang.org/x/net/context"
	"github.com/CyCoreSystems/ari/v5"
	//"github.com/CyCoreSystems/ari/v5/client/native"
	"github.com/CyCoreSystems/ari/v5/rid"
	"google.golang.org/grpc/metadata"
	"github.com/rotisserie/eris"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/api"
	"lineblocs.com/processor/mngrs"
)

type Server struct {
	Client ari.Client
	WsChan chan<- *ClientEvent
}
func (s *Server) lookupBridge( bridgeId string ) (*types.LineBridge, error) {
	src := ari.NewKey(ari.BridgeKey, bridgeId)
	bridge := s.Client.Bridge().Get(src)

	return &types.LineBridge{ Bridge: bridge }, nil
}

func (s *Server) lookupChannel( channelId string ) (*types.LineChannel, error) {
	src := ari.NewKey(ari.ChannelKey, channelId)
	channel := s.Client.Channel().Get(src)

	return &types.LineChannel{ Channel: channel }, nil
}
func (s *Server) safeSendToWS( clientid string, evt *ClientEvent ) {
	wsChan := lookupWSChan( clientid )
	if wsChan != nil {
		fmt.Println("sending back to WS")
		wsChan <- evt
		return
	}
	fmt.Println("could not find WS to send to")
}
func (s *Server) startProcessingBridge( bridge *types.LineBridge, clientId string ) () {
	h := bridge.Bridge
	// Delete the bridge when we exit
	defer h.Delete()

	destroySub := h.Subscribe(ari.Events.BridgeDestroyed)
	defer destroySub.Cancel()

	enterSub := h.Subscribe(ari.Events.ChannelEnteredBridge)
	defer enterSub.Cancel()

	leaveSub := h.Subscribe(ari.Events.ChannelLeftBridge)
	defer leaveSub.Cancel()

	for {
		select {
		case <-destroySub.Events():
			fmt.Println("bridge destroyed")
			return
		case e, ok := <-enterSub.Events():
			if !ok {
				fmt.Println("channel entered subscription closed")
				return
			}
			fmt.Println("channel joined bridge!!")
			v := e.(*ari.ChannelEnteredBridge)
			channelId := v.Channel.ID
			s.dispatchEvent(func() {
				// send to channel
				data := make(map[string]string)
				data["bridge_id"] = h.ID()
				data["channel_id"] = channelId
				//data["channel_id"] = req.ChannelId
				evt:= ClientEvent{
					ClientId: clientId,
					Type: "bridge_ChannelJoined",
					Data: data }

				fmt.Println("sending client event..")
				s.safeSendToWS( clientId, &evt )
			})
		case e, ok := <-leaveSub.Events():
			if !ok {
				return
			}
			v := e.(*ari.ChannelLeftBridge)
			channelId := v.Channel.ID
			s.dispatchEvent(func() {
				// send to channel
				data := make(map[string]string)
				data["bridge_id"] = h.ID()
				data["channel_id"] = channelId
				//data["channel_id"] = req.ChannelId
				evt:= ClientEvent{
					ClientId: clientId,
					Type: "bridge_ChannelLeft",
					Data: data }
				fmt.Println("sending client event..")
				s.safeSendToWS( clientId, &evt )
			})
		}
	}
}

func (s *Server) addBridgeChannel( bridge *types.LineBridge, channel *types.LineChannel ) (error) {
	err := bridge.Bridge.AddChannel(channel.Channel.Key().ID)
	if err != nil {
		return err
	}
	utils.AddChannelToBridge(bridge, channel)
	return nil
}

func (s *Server) manageCall( call *types.Call, callChannel *types.LineChannel, clientId string, ringTimeoutChan chan<- bool) () {
	h := callChannel.Channel
	// Delete the bridge when we exit
	endSub := h.Subscribe(ari.Events.StasisEnd)
	defer endSub.Cancel()
	startSub := h.Subscribe(ari.Events.StasisStart)
	defer startSub.Cancel()
	destroySub := h.Subscribe(ari.Events.ChannelDestroyed)
	defer destroySub.Cancel()
	dtmfSub := h.Subscribe(ari.Events.ChannelDtmfReceived)
	defer dtmfSub.Cancel()
	dtmfGathered := ""
	h.Answer()

	for {
		select {
		case <-destroySub.Events():
			fmt.Println("channel destroyed 111")
			return
		case e := <-dtmfSub.Events():
			v := e.(*ari.ChannelDtmfReceived)
			digit := v.Digit
			fmt.Println("input received DTMF: " + digit)
			dtmfGathered = dtmfGathered + digit
			s.dispatchEvent(func() {
				// send to channel
				data := make(map[string]string)
				data["channel_id"] = callChannel.Channel.ID()
				data["dtmf_gathered"] = dtmfGathered
				data["digit"] = digit
				//data["channel_id"] = req.ChannelId
				evt:= ClientEvent{
					ClientId: clientId,
					Type: "channel_DTMFReceived",
					Data: data }
				fmt.Println("sending client event..")
				s.safeSendToWS( clientId, &evt )
			})
		case e := <-startSub.Events():

			fmt.Println("channel started")
			v := e.(*ari.StasisStart)
			channelId := v.Channel.ID
			ringTimeoutChan <- true
			s.dispatchEvent(func() {
				// send to channel
				data := make(map[string]string)
				data["channel_id"] = channelId
				data["call_id"] = strconv.Itoa( call.CallId )
				//data["channel_id"] = req.ChannelId
				evt:= ClientEvent{
					ClientId: clientId,
					Type: "channel_ChannelStart",
					Data: data }
				fmt.Println("sending client event..")
				s.safeSendToWS( clientId, &evt )
			})
		case e, ok := <-endSub.Events():
			if !ok {
				return
			}
			fmt.Println("channel ended")
			v := e.(*ari.StasisEnd)
			channelId := v.Channel.ID
			s.dispatchEvent(func() {
				// send to channel
				data := make(map[string]string)
				data["bridge_id"] = h.ID()
				data["channel_id"] = channelId
				//data["channel_id"] = req.ChannelId
				evt:= ClientEvent{
					ClientId: clientId,
					Type: "channel_ChannelEnd",
					Data: data }

				fmt.Println("sending client event..")
				s.safeSendToWS( clientId, &evt )
			})
		}
	}
}

func (s *Server) managePrompt(playback *ari.PlaybackHandle, clientId string) {
	finishedSub := playback.Subscribe(ari.Events.PlaybackFinished)
	defer finishedSub.Cancel()

	fmt.Println("waiting for playback to finish...")
	for {
		select {
		case <-finishedSub.Events():
			fmt.Println("playback finished...")
			s.dispatchEvent(func() {
				// send to channel
				data := make(map[string]string)
				data["playback_id"] = playback.ID()
				evt:= ClientEvent{
					ClientId: clientId,
					Type: "playback_PlaybackFinished",
					Data: data }
					fmt.Println("sending client event..")
				s.safeSendToWS( clientId, &evt )
			})
			return
		}
	}
}

func (s *Server) startListeningForRingTimeout(timeout int, channel *types.LineChannel, ringTimeoutChan <-chan bool) {
    duration := time.Now().Add(time.Duration( timeout ) * time.Second)

    // Create a context that is both manually cancellable and will signal
    // a cancel at the specified duration.
    ringCtx, cancel := context.WithDeadline(context.Background(), duration)
    defer cancel()
	for {
    select {
		case <-ringTimeoutChan:
				fmt.Println("bridge in session. stopping ring timeout")
				return
			case <-ringCtx.Done():
				fmt.Println("Ring timeout elapsed.. ending all calls")
				utils.SafeHangup( channel )
				return
    }
}
}



func NewServer(cl ari.Client, wsChan chan<- *ClientEvent) (*Server) {
	srv := Server{
		Client: cl,
		WsChan: wsChan }
		return &srv
}

type event func()
func (s *Server) dispatchEvent( handler event ) {
	go func() {
		time.Sleep(time.Duration(1000) * time.Millisecond)


		// call the handler
		handler()
	}()
}

func (s *Server) CreateBridge(ctx context.Context, req *BridgeRequest) (*BridgeReply, error) {
	fmt.Println("creating bridge!!!");
	//var bridge *ari.BridgeHandle 
	var err error
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	fmt.Println("client ID = " + clientId)
	src := ari.NewKey(ari.ChannelKey, "123")
	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
	bridge, err := s.Client.Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		return nil, eris.Wrap(err, "failed to create bridge")
	}

	reply := BridgeReply{
		BridgeId: key.ID }

	s.dispatchEvent(func() {
		// send to channel
		data := make(map[string]string)
		data["bridge_id"] = key.ID
		evt:= ClientEvent{
			ClientId: clientId,
			Type: "bridge_BridgeCreated",
			Data: data }
			fmt.Println("sending client event..")
		s.safeSendToWS( clientId, &evt )
	})

	lineBridge := types.LineBridge{
		Bridge: bridge }
	go s.startProcessingBridge( &lineBridge, clientId )
	return &reply, nil
}
func (s *Server) CreateCall(ctx context.Context, req *CallRequest) (*CallReply, error) {
	//var bridge *ari.BridgeHandle 
	var err error
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	workspaceId := headers["workspaceid"][0]
	userId := headers["userid"][0]
	domain := headers["domain"][0]
	fmt.Println("client ID = " + clientId)

	callerId := req.CallerId
	fmt.Println("caller ID was set to: " + callerId)

	valid, err := api.VerifyCallerId(workspaceId, callerId)
	if err != nil {
		fmt.Println("verify error: " + err.Error())
		return nil, err
	}
	if !valid {
		fmt.Println("caller id was invalid. user provided: " + callerId)
		return nil, err
	}

	numberToCall := req.Destination
	//key := src.New(ari.ChannelKey, rid.New(rid.Channel))

	fmt.Println("Calling: " + numberToCall)

	timeout := utils.ParseRingTimeout( req.Timeout )

	outChannel := types.LineChannel{}
	outboundChannel, err := s.Client.Channel().Create(nil, utils.CreateChannelRequest( numberToCall )	)

	if err != nil {
		fmt.Println("error creating outbound channel: " + err.Error())
		return nil, err
	}

	callType :=  req.CallType

	sipHeaders := utils.CreateSIPHeaders(domain, callerId, callType)
	outboundChannel, err = outboundChannel.Originate( utils.CreateOriginateRequest(callerId, numberToCall, sipHeaders) )

	if err != nil {
		fmt.Println( "error occured: " + err.Error() )
		return nil, err
	}
	outChannel.Channel = outboundChannel

	user, err := strconv.Atoi( userId )
	if err != nil {
		fmt.Println( "error occured: " + err.Error() )
		return nil, err
	}

	workspace, err := strconv.Atoi( workspaceId )
	if err != nil {
		fmt.Println( "error occured: " + err.Error() )
		return nil, err
	}
	params := types.CallParams{
		From: callerId,
		To: numberToCall,
		Status: "start",
		Direction: "outbound",
		UserId:  user,
		WorkspaceId: workspace,
		ChannelId: outboundChannel.ID() }
	body, err := json.Marshal( params )
	if err != nil {
		fmt.Println( "error occured: " + err.Error() )
		return nil, err
	}

	fmt.Println("creating outbound call...")
	resp, err := api.SendHttpRequest( "/call/createCall", body )
	call, err := utils.CreateCall( resp.Headers.Get("x-call-id"), &outChannel, &params)

	if err != nil {
		fmt.Println( "error occured: " + err.Error() )
		return nil, err
	}
	stopChannel := make( chan bool, 1 )
	go s.manageCall( call, &outChannel, clientId, stopChannel )
	go s.startListeningForRingTimeout(timeout, &outChannel, stopChannel)
	reply := CallReply{
		ChannelId: outChannel.Channel.ID(),
		CallId: strconv.Itoa( call.CallId ) }
	return &reply, nil
}
func (*Server) AddChannel(context.Context, *ChannelRequest) (*ChannelReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddChannel not implemented")
}
func (*Server) PlayRecording(context.Context, *RecordingPlayRequest) (*RecordingPlayReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PlayRecording not implemented")
}
func (*Server) GetChannel(context.Context, *ChannelFetchRequest) (*ChannelFetchReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetChannel not implemented")
}
func (s *Server) CreateConference(ctx context.Context, req *ConferenceRequest) (*ConferenceReply, error) {
	fmt.Println("creating conf!!!");
	//var bridge *ari.BridgeHandle 
	var err error
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	workspaceId := headers["workspaceid"][0]
	userId := headers["userid"][0]
	domain := headers["domain"][0]
	fmt.Println("client ID = " + clientId)
	workspaceName := utils.GetWorkspaceNameFromDomain( domain )
	userIdInt, err := strconv.Atoi( userId )
	if err != nil {
		fmt.Println("startExecution err " + err.Error())
		return nil, err
	}
	workspace, err := strconv.Atoi( workspaceId )
	if err != nil {
		fmt.Println("startExecution err " + err.Error())
		return nil, err
	}

	user := types.NewUser(userIdInt, workspace, workspaceName)
	resp, err := api.CreateConference( workspace, req.Name)
	if err != nil {
		return nil,err
	}

	src := ari.NewKey(ari.ChannelKey, "123")
	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
	bridge, err := s.Client.Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		return nil, err
	}


	conf := types.NewConference(resp.Id, user, &types.LineBridge{Bridge:bridge})


	utils.AddConfBridge( s.Client, workspaceId, req.Name, conf )

	reply := ConferenceReply{
		ConfId: conf.Id,
		BridgeId: conf.Bridge.Bridge.ID() }

	s.dispatchEvent(func() {
		// send to channel
		data := make(map[string]string)
		data["conf_id"] = resp.Id
		evt:= ClientEvent{
			ClientId: clientId,
			Type: "conference_ConfCreated",
			Data: data }
			fmt.Println("sending client event..")
		s.safeSendToWS( clientId, &evt )
	})
	return &reply, nil
}
func (*Server) ChannelGetBridge(context.Context, *ChannelGetBridgeRequest) (*ChannelGetBridgeReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelGetBridge not implemented")
}
func (s *Server) ChannelRemoveFromBridge(ctx context.Context, req *ChannelRemoveBridgeRequest) (*ChannelRemoveBridgeReply, error) {
	fmt.Println("adding channel to bridge!!!");
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	fmt.Println("client ID = " + clientId)
	bridge, err := s.lookupBridge( req.BridgeId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}

	channel, err := s.lookupChannel( req.ChannelId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}

	err = bridge.Bridge.RemoveChannel( channel.Channel.ID() )
	if err != nil {
		return nil, eris.Wrap(err, "failed to remove channel from bridge")
	}
	resp := ChannelRemoveBridgeReply{}
	return &resp, nil
}
func (s *Server) ChannelPlayTTS(ctx context.Context, req *ChannelTTSRequest) (*ChannelTTSReply, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]

	fmt.Println("client ID = " + clientId)
	fmt.Println("lookup channel = " + req.ChannelId)
	channel, err := s.lookupChannel( req.ChannelId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}

	file, err := utils.StartTTS(req.Text, req.Gender, req.Voice, req.Language)
	if err != nil {
		fmt.Println("error downloading: " + err.Error())
		return nil, err
	}

	uniq, err := uuid.NewUUID()
	if err != nil {
		fmt.Println("error creating UUID: " + err.Error())
		return nil, err
	}
	key := ari.NewKey(ari.PlaybackKey, uniq.String())
	uri := "sound:" + file
	playback, err := channel.Channel.Play(key.ID, uri)
	//playback, err := channel.Channel.Play(channel.Channel.Key().ID, uri)
	if err != nil {
		fmt.Println("failed to play join sound. err: " + err.Error())
		return nil, err
	}

	go s.managePrompt( playback, clientId )
	reply := ChannelTTSReply{
		PlaybackId: key.ID }
	return &reply, nil
}
func (*Server) ChannelStartAcceptingInput(context.Context, *ChannelInputRequest) (*ChannelInputReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelStartAcceptingInput not implemented")
}
func (s *Server) ChannelRemoveDTMFListeners(ctx context.Context, req *ChannelRemoveDTMFRequest) (*ChannelRemoveDTMFReply, error) {
	fmt.Println("remove DTMF listeners")
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	fmt.Println("client ID = " + clientId)
	channel, err := s.lookupChannel( req.ChannelId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}

	channel.Channel.Unsubscribe(ari.Events.ChannelDtmfReceived)
	resp := ChannelRemoveDTMFReply{}
	return &resp, nil
}
func (*Server) ChannelAutomateCallHangup(context.Context, *GenericChannelReq) (*GenericChannelResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelAutomateCallHangup not implemented")
}
func (*Server) ChannelGotoFlowWidget(context.Context, *ChannelFlowWidgetRequest) (*ChannelFlowWidgetReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelGotoFlowWidget not implemented")
}

func (s *Server) ChannelStartFlow(ctx context.Context, req *ChannelStartFlowWidgetRequest) (*ChannelStartFlowWidgetReply, error) {
	var err error
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	workspaceId := headers["workspaceid"][0]
	userId := headers["userid"][0]
	domain := headers["domain"][0]
	workspaceName := utils.GetWorkspaceNameFromDomain( domain )
	fmt.Println("client ID = " + clientId)
	channel, err := s.lookupChannel( req.ChannelId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}
	flowData, err := api.GetFlowInfo(workspaceId, req.FlowId)
	if err != nil {
		fmt.Println("error starting new flow: " + err.Error())
		return nil, err
	}
	info := flowData.Vars
	//user := types.NewUser(data.CreatorId, data.WorkspaceId, data.WorkspaceName)

	vals := make(map[string]string)
	vals["workspace"] = workspaceId
	body, err := api.SendGetRequest("/user/getWorkspaceMacros", vals)

	if err != nil {
		fmt.Println("startExecution err " + err.Error())
		return nil, err
	}
	var macros []*types.WorkspaceMacro
	err = json.Unmarshal( []byte(body), &macros)
	if err != nil {
		fmt.Println("startExecution err " + err.Error())
		return nil, err
	}

	userIdInt, err := strconv.Atoi( userId )
	if err != nil {
		fmt.Println("startExecution err " + err.Error())
		return nil, err
	}
	workspace, err := strconv.Atoi( workspaceId )
	if err != nil {
		fmt.Println("startExecution err " + err.Error())
		return nil, err
	}
	
	user := types.NewUser(userIdInt, workspace, workspaceName)
	flow := types.NewFlow(
		flowData.FlowId,	
		user,
		info,
		channel, 
		macros,
		s.Client)

	vars := make( map[string] string )
	flowCtx, _ := context.WithCancel(context.Background())
	go mngrs.ProcessFlow( s.Client, flowCtx, flow, channel, vars, flow.Cells[ 0 ])
	resp := ChannelStartFlowWidgetReply{}
	return &resp, nil
}
func (s *Server) ChannelStartRinging(ctx context.Context, req *GenericChannelReq) (*GenericChannelResp, error) {
	fmt.Println("remove DTMF listeners")
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	fmt.Println("client ID = " + clientId)
	channel, err := s.lookupChannel( req.ChannelId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}

	channel.Channel.Ring()
	resp := GenericChannelResp{}
	return &resp, nil
}
func (s *Server) ChannelStopRinging(ctx context.Context, req *GenericChannelReq) (*GenericChannelResp, error) {
	fmt.Println("remove DTMF listeners")
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	fmt.Println("client ID = " + clientId)
	channel, err := s.lookupChannel( req.ChannelId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}

	channel.Channel.StopRing()
	resp := GenericChannelResp{}
	return &resp, nil
}
func (s *Server) ChannelHangup(ctx context.Context, req *GenericChannelReq) (*GenericChannelResp, error) {
	fmt.Println("remove DTMF listeners")
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	fmt.Println("client ID = " + clientId)
	channel, err := s.lookupChannel( req.ChannelId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}

	channel.Channel.Hangup()
	resp := GenericChannelResp{}
	return &resp, nil
}
func (s *Server) BridgeAddChannel(ctx context.Context, req *BridgeChannelRequest) (*BridgeChannelReply, error) {
	fmt.Println("adding channel to bridge!!!");
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	fmt.Println("client ID = " + clientId)
	bridge, err := s.lookupBridge( req.BridgeId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}

	channel, err := s.lookupChannel( req.ChannelId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}
	fmt.Println("adding channel now!!!");

	s.addBridgeChannel( bridge, channel )
	reply := BridgeChannelReply{}
	return &reply, nil
}
func (s *Server) BridgeAddChannels(ctx context.Context, req *BridgeChannelsRequest) (*BridgeChannelsReply, error) {
	fmt.Println("adding channel to bridge!!!");
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	fmt.Println("client ID = " + clientId)


	bridge, err := s.lookupBridge( req.BridgeId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}

	for _, channelId := range req.ChannelId {
		channel, err := s.lookupChannel( channelId )
		if err != nil {
			return nil, eris.Wrap(err, "failed to add channel to bridge")
		}
		s.addBridgeChannel( bridge, channel )
	}
	reply := BridgeChannelsReply{}
	return &reply, nil
}
func (s *Server) BridgePlayTTS(ctx context.Context, req *BridgeTTSRequest) (*BridgeTTSReply, error) {
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]

	fmt.Println("client ID = " + clientId)
	fmt.Println("lookup bridge  = " + req.BridgeId)
	bridge, err := s.lookupBridge( req.BridgeId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}

	file, err := utils.StartTTS(req.Text, req.Gender, req.Voice, req.Language)
	if err != nil {
		fmt.Println("error downloading: " + err.Error())
		return nil, err
	}

	uniq, err := uuid.NewUUID()
	if err != nil {
		fmt.Println("error creating UUID: " + err.Error())
		return nil, err
	}
	key := ari.NewKey(ari.PlaybackKey, uniq.String())
	uri := "sound:" + file
	playback, err := bridge.Bridge.Play(key.ID, uri)
	//playback, err := channel.Channel.Play(channel.Channel.Key().ID, uri)
	if err != nil {
		fmt.Println("failed to play join sound. err: " + err.Error())
		return nil, err
	}

	go s.managePrompt( playback, clientId )
	reply := BridgeTTSReply{
		PlaybackId: key.ID }
	return &reply, nil
}
func (*Server) BridgeAutomateLegAHangup(context.Context, *BridgeAutomateLegRequest) (*BridgeAutomateLegReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BridgeAutomateLegAHangup not implemented")
}
func (*Server) BridgeAutomateLegBHangup(context.Context, *BridgeAutomateLegRequest) (*BridgeAutomateLegReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BridgeAutomateLegBHangup not implemented")
}
func (*Server) BridgeAttachEventListener(context.Context, *BridgeEventRequest) (*BridgeEventReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BridgeAttachEventListener not implemented")
}
func (*Server) RecordingDeleteRecording(context.Context, *RecordingDeleteRequest) (*RecordingDeleteReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RecordingDeleteRecording not implemented")
}
func (*Server) RecordingAddRecordingTag(context.Context, *RecordingTagRequest) (*RecordingTagReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RecordingAddRecordingTag not implemented")
}
func (*Server) RecordingDeleteRecordingTag(context.Context, *RecordingTagDeleteRequest) (*RecordingTagDeleteReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RecordingDeleteRecordingTag not implemented")
}
func (*Server) SessionListRecordings(context.Context, *SessionRecordingsRequest) (*SessionRecordingsReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SessionListRecordings not implemented")
}
func (*Server) ConferenceAddWaitingParticipant(context.Context, *ConferenceParticipantRequest) (*ConferenceParticipantReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConferenceAddWaitingParticipant not implemented")
}
func (*Server) ConferenceAddParticipant(context.Context, *ConferenceParticipantRequest) (*ConferenceParticipantReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConferenceAddParticipant not implemented")
}
func (*Server) ConferenceSetModeratorInConf(context.Context, *ConferenceModeratorRequest) (*ConferenceModeratorReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConferenceSetModeratorInConf not implemented")
}
func (*Server) ConferenceAttachEventListener(context.Context, *ConferenceEventRequest) (*ConferenceEventReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConferenceAttachEventListener not implemented")
}

func (s *Server) BridgeDestroy(ctx context.Context, req *GenericBridgeReq) (*GenericBridgeResp, error) {
	fmt.Println("destroy bridge")
	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("could not get metadata")
	}
	clientId := headers["clientid"][0]
	fmt.Println("client ID = " + clientId)
	bridge, err := s.lookupBridge( req.BridgeId )
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}
	//src := ari.NewKey(ari.BridgeKey, bridge.Bridge.ID())
	//key := src.New(ari.BridgeKey, rid.New(rid.Bridge))

	data, err := bridge.Bridge.Data()
	if err != nil {
		return nil, eris.Wrap(err, "failed to add channel to bridge")
	}
	for _, id := range data.ChannelIDs {
		channel, err := s.lookupChannel( id )
		if err != nil {
			//return nil, eris.Wrap(err, "failed to add channel to get channel")
			continue
		}
		channel.Channel.Hangup()
	}
	//bridge.EndBridgeCall()
	resp := GenericBridgeResp{}
	return &resp, nil
}