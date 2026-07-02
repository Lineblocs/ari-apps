package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	helpers "github.com/Lineblocs/go-helpers"
	_ "github.com/joho/godotenv"
	"github.com/sirupsen/logrus"

	"github.com/CyCoreSystems/ari-proxy/v5/client"
	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/client/native"
	"lineblocs.com/processor/api"
	"lineblocs.com/processor/grpc"
	"lineblocs.com/processor/logger"
	"lineblocs.com/processor/mngrs"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/resources"
)

var ariApp = "lineblocs"

var bridge *ari.BridgeHandle


type APIResponse struct {
	Headers http.Header
	Body    []byte
}

func createARIConnection(connectCtx context.Context) (ari.Client, error) {
	var err error
	var cl ari.Client
	var useProxy bool
	host := os.Getenv("ARI_HOST")
	ariUrl := fmt.Sprintf("http://%s:8088/ari", host)
	wsUrl := fmt.Sprintf("ws://%s:8088/ari/events", host)
	helpers.Log(logrus.InfoLevel, "Connecting to: "+ariUrl)
	proxy := os.Getenv("ARI_USE_PROXY")
	if proxy != "" {
		useProxy, err = strconv.ParseBool(proxy)
		if err != nil {
			return nil, err
		}
	}
	ctx := context.Background()
	if useProxy {
		helpers.Log(logrus.DebugLevel, "Using ARI proxy!!!")
		natsgw := os.Getenv("NATSGW_URL")
		cl, err := client.New(ctx,
			client.WithApplication(ariApp),
			client.WithURI(natsgw))
		return cl, err
	}
	helpers.Log(logrus.InfoLevel, "Directly connecting to ARI server\r\n")
	cl, err = native.Connect(&native.Options{
		Application:  ariApp,
		Username:     os.Getenv("ARI_USERNAME"),
		Password:     os.Getenv("ARI_PASSWORD"),
		URL:          ariUrl,
		WebsocketURL: wsUrl})
	return cl, err
}

func startProcessingWSEvents() {
	defer func() {
		if r := recover(); r != nil {
			helpers.Log(logrus.ErrorLevel, fmt.Sprintf("PANIC in startProcessingWSEvents recovered: %v. Restarting...", r))
			time.Sleep(time.Second * 5)
			startProcessingWSEvents()
		}
	}()

	retryCount := 0
	maxRetries := 10
	backoffDuration := time.Second

	for {
		helpers.Log(logrus.InfoLevel, "Connecting to ARI (attempt %d)", retryCount+1)
		connectCtx, cancel := context.WithCancel(context.Background())
		cl, err := createARIConnection(connectCtx)

		if err != nil {
			retryCount++
			helpers.Log(logrus.ErrorLevel, "could not connect to ARI. error: %s. Retrying in %v...", err.Error(), backoffDuration)
			cancel()
			if retryCount < maxRetries {
				time.Sleep(backoffDuration)
				if backoffDuration < time.Minute {
					backoffDuration *= 2
				}
				continue
			} else {
				helpers.Log(logrus.ErrorLevel, "Max retries reached. Exiting.")
				return
			}
		}

	retryCount = 0
	backoffDuration = time.Second

	helpers.Log(logrus.InfoLevel, "Connected to ARI")

	defer func() {
		helpers.Log(logrus.InfoLevel, "Closing ARI connection")
		cl.Close()
		cancel()
	}()

	helpers.Log(logrus.InfoLevel, "starting GRPC listener...")
	go grpc.StartListener(cl)

	helpers.Log(logrus.InfoLevel, "Starting listener app")
	helpers.Log(logrus.InfoLevel, "Listening for new calls")
	sub := cl.Bus().Subscribe(nil, "StasisStart")

	reconnectTicker := time.NewTicker(time.Second * 30)
	defer reconnectTicker.Stop()

	for {
		if !cl.Connected() {
			helpers.Log(logrus.ErrorLevel, "websocket was disconnected. Reconnecting...")
			cl.Close()
			cancel()
			sub.Cancel()
			break
		}

		select {
		case e := <-sub.Events():
			if e == nil {
				helpers.Log(logrus.WarnLevel, "Received nil event from subscription")
				break
			}
			if ss, ok := e.(*ari.StasisStart); ok {
				helpers.Log(logrus.InfoLevel, "Got stasis start for channel: %s", ss.Channel.ID)
				go func(ssEvent *ari.StasisStart) {
					defer func() {
						if r := recover(); r != nil {
							helpers.Log(logrus.ErrorLevel, fmt.Sprintf("PANIC in startExecution recovered: %v for channel %s", r, ssEvent.Channel.ID))
						}
					}()
					startExecution(cl, ssEvent, cl.Channel().Get(ssEvent.Key(ari.ChannelKey, ssEvent.Channel.ID)))
				}(ss)
			} else {
				helpers.Log(logrus.WarnLevel, "Unexpected event type received: %T", e)
			}
		case <-reconnectTicker.C:
			if !cl.Connected() {
				helpers.Log(logrus.WarnLevel, "Connection health check failed. Reconnecting...")
				cl.Close()
				cancel()
				sub.Cancel()
				break
			}
		case <-connectCtx.Done():
			helpers.Log(logrus.InfoLevel, "Context cancelled. Closing connection.")
			cl.Close()
			sub.Cancel()
			return
		}
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("FATAL PANIC in main: %v\n", r)
			os.Exit(1)
		}
	}()

	// OPTIONAL: setup logging
	//native.Logger = log
	// Init Logrus and configure channels
	logDestination := utils.Config("LOG_DESTINATIONS")
	helpers.InitLogrus(logDestination)
	helpers.Log(logrus.InfoLevel, "Starting ARI application service")

	for {
		startProcessingWSEvents()
		helpers.Log(logrus.InfoLevel, "Restarting main event processing loop...")
		time.Sleep(time.Second * 5)
	}
}

type bridgeManager struct {
	h *ari.BridgeHandle
}

func createCall() (types.Call, error) {
	return types.Call{}, nil
}
func createCallDebit(user *types.User, call *types.Call, direction string) error {
	return nil
}
func attachChannelLifeCycleListeners(flow *types.Flow, channel *types.LineChannel, callChannel chan *types.Call) {
	defer func() {
		if r := recover(); r != nil {
			helpers.Log(logrus.ErrorLevel, fmt.Sprintf("PANIC in attachChannelLifeCycleListeners recovered: %v", r))
		}
	}()

	var call *types.Call
	endSub := channel.Channel.Subscribe(ari.Events.StasisEnd)
	defer endSub.Cancel()

	call = nil

	for {

		select {
		case <-endSub.Events():
			helpers.Log(logrus.DebugLevel, "stasis end called")
			if call == nil {
				helpers.Log(logrus.WarnLevel, "Call is nil when stasis end received")
				break
			}
			call.Ended = time.Now()
			params := types.StatusParams{
				CallId: call.CallId,
				Ip:     utils.GetPublicIp(),
				Status: "ENDED"}
			body, err := json.Marshal(params)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "JSON marshalling error: %s for call %d", err.Error(), call.CallId)
				continue
			}

			_, err = api.SendHttpRequest("/call/updateCall", body)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "Failed to update call status: %s for call %d", err.Error(), call.CallId)
				continue
			}
			err = createCallDebit(flow.User, call, "INBOUND")
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "Failed to create call debit: %s for call %d", err.Error(), call.CallId)
				continue
			}
			helpers.Log(logrus.InfoLevel, "Call %d ended successfully", call.CallId)

		case call = <-callChannel:
			if call == nil {
				helpers.Log(logrus.WarnLevel, "Received nil call from channel")
				break
			}
			helpers.Log(logrus.DebugLevel, "call is setup with ID: %d", call.CallId)
		}
	}
}
func attachDTMFListeners(channel *types.LineChannel) {
	defer func() {
		if r := recover(); r != nil {
			helpers.Log(logrus.ErrorLevel, fmt.Sprintf("PANIC in attachDTMFListeners recovered: %v", r))
		}
	}()

	dtmfSub := channel.Channel.Subscribe(ari.Events.ChannelDtmfReceived)
	defer dtmfSub.Cancel()

	for {

		select {
		case <-dtmfSub.Events():
			helpers.Log(logrus.DebugLevel, "Received DTMF on channel %s", channel.Channel.ID())
		}
	}
}

func processIncomingCall(cl ari.Client, flow *types.Flow, lineChannel *types.LineChannel, exten string, callerId string, sipCallId string) {
	defer func() {
		if r := recover(); r != nil {
			helpers.Log(logrus.ErrorLevel, fmt.Sprintf("PANIC in processIncomingCall recovered: %v for exten %s caller %s", r, exten, callerId))
			if lineChannel != nil && lineChannel.Channel != nil {
				defer func() {
					if rr := recover(); rr != nil {
						helpers.Log(logrus.ErrorLevel, fmt.Sprintf("Error hangup channel: %v", rr))
					}
				}()
				lineChannel.SafeHangup()
			}
		}
	}()

	go attachDTMFListeners(lineChannel)
	callChannel := make(chan *types.Call)
	go attachChannelLifeCycleListeners(flow, lineChannel, callChannel)

	helpers.Log(logrus.DebugLevel, "Calling API to create call...")
	helpers.Log(logrus.DebugLevel, "exten is: %s", exten)
	helpers.Log(logrus.DebugLevel, "caller ID is: %s", callerId)
	helpers.Log(logrus.DebugLevel, "SIP call id: %s", sipCallId)
	params := types.CallParams{
		From:        callerId,
		To:          exten,
		Status:      "STARTED",
		Direction:   "INBOUND",
		UserId:      flow.User.Id,
		WorkspaceId: flow.User.Workspace.Id,
		ChannelId:   lineChannel.Channel.ID(),
		SIPCallId: sipCallId,
	}
	body, err := json.Marshal(params)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return
	}

	helpers.Log(logrus.InfoLevel, "creating call...")
	resp, err := api.SendHttpRequest("/call/createCall", body)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return
	}

	id := resp.Headers.Get("x-call-id")
	helpers.Log(logrus.DebugLevel, "Call ID is: "+id)
	idAsInt, err := strconv.Atoi(id)

	call := types.Call{
		CallId:  idAsInt,
		Channel: lineChannel,
		Started: time.Now(),
		Params:  &params}

	flow.RootCall = &call
	helpers.Log(logrus.DebugLevel, "answering call..")
	lineChannel.Answer()
	vars := make(map[string]string)
	go mngrs.ProcessFlow(cl, flow, lineChannel, vars, flow.Cells[0])
	callChannel <- &call
}

func startExecution(cl ari.Client, event *ari.StasisStart, h *ari.ChannelHandle) {
	defer func() {
		if r := recover(); r != nil {
			helpers.Log(logrus.ErrorLevel, fmt.Sprintf("PANIC in startExecution recovered for channel %s: %v", h.Key().ID, r))
			if h != nil {
				defer func() {
					if rr := recover(); rr != nil {
						helpers.Log(logrus.ErrorLevel, fmt.Sprintf("Error hangup channel: %v", rr))
					}
				}()
				h.Hangup()
			}
		}
	}()

	helpers.Log(logrus.InfoLevel, "running app for channel %s", h.Key().ID)

	if len(event.Args) < 2 {
		helpers.Log(logrus.ErrorLevel, "Invalid StasisStart event: insufficient arguments for channel %s", h.Key().ID)
		if h != nil {
			h.Hangup()
		}
		return
	}

	action := event.Args[0]
	exten := event.Args[1]
	vals := make(map[string]string)
	vals["number"] = exten

	helpers.Log(logrus.DebugLevel, "received action: %s for channel %s", action, h.Key().ID)
	helpers.Log(logrus.DebugLevel, "EXTEN: %s", exten)

	switch action {
	case "h":
		fmt.Println("Received h handler - not processing")
	case "PROCESSED_CALL":
		fmt.Println("Already dialed - not processing")
		return
	case "INCOMING_SIP_TRUNK":
		//domain := data.Domain
		exten := event.Args[1]
		callerId := event.Args[2]
		trunkAddr := event.Args[3]
		lineChannel := types.NewChannel(h, true)
		lineChannel.Answer()

		resp, err := api.GetUserByDID(exten)
		helpers.Log(logrus.DebugLevel, "exten ="+exten)
		helpers.Log(logrus.DebugLevel, "caller ID ="+callerId)
		helpers.Log(logrus.DebugLevel, "trunk addr ="+trunkAddr)
		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not get domain. error: "+err.Error())
			return
		}
		helpers.Log(logrus.DebugLevel, "workspace ID= "+strconv.Itoa(resp.WorkspaceId))
		user := types.NewUser(resp.Id, resp.WorkspaceId, resp.WorkspaceName)
		err = utils.ProcessSIPTrunkCall(cl, lineChannel.Channel.Key(), user, &lineChannel, callerId, exten, trunkAddr)
		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not create bridge. error: "+err.Error())
			lineChannel.SafeHangup()
			return

		}

	case "INCOMING_CALL":
		body, err := api.SendGetRequest("/user/getDIDNumberData", vals)

		if err != nil {
			helpers.Log(logrus.ErrorLevel, "startExecution err "+err.Error())
			return
		}

		var data types.FlowDIDData
		var flowJson types.FlowVars
		err = json.Unmarshal([]byte(body), &data)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "startExecution err "+err.Error())
			return
		}

		if utils.CheckFreeTrial(data.Plan) {
			helpers.Log(logrus.ErrorLevel, "Ending call due to free trial")
			h.Hangup()
			helpers.Log(logrus.DebugLevel, fmt.Sprintf("msg = %s", logger.FREE_TRIAL_ENDED))
			return
		}

		// Corrected and cleaner Go switch case
		switch {
		case data.FlowJson == "" && data.CreationIntent == "CREATED_WITH_DID_PURCHASE":
			// This case handles the original 'if' block: Do nothing and continue.
			// The code block is intentionally empty, just like the original 'if' block.
			err = json.Unmarshal([]byte(resources.DIDFlowUnconfiguredJSON), &flowJson)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "startExecution err "+err.Error())
				return
			}
		default:
			// The 'default' case handles the original 'else' block. 
			// It runs if the 'case' condition above is FALSE, meaning:
			// (data.FlowJson != "" || data.CreationIntent != "CREATED_WITH_DID_PURCHASE")
			
			// We only need data.FlowJson != "" for the Unmarshal, but the logic 
			// must be the inverse of the 'case' above to be a true replacement for the 'else'.
			
			err = json.Unmarshal([]byte(data.FlowJson), &flowJson)
			if err != nil {
				helpers.Log(logrus.ErrorLevel, "startExecution err "+err.Error())
				return
			}
		}

		fmt.Printf("got %d models in data\r\n", len(flowJson.Models))
		body, err = api.SendGetRequest("/user/getWorkspaceMacros", vals)

		if err != nil {
			helpers.Log(logrus.ErrorLevel, "startExecution err "+err.Error())
			return
		}
		var macros []*types.WorkspaceMacro
		err = json.Unmarshal([]byte(body), &macros)
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "startExecution err "+err.Error())
			return
		}

		lineChannel := types.NewChannel(h, true)
		user := types.NewUser(data.CreatorId, data.WorkspaceId, data.WorkspaceName)
		flow := types.NewFlow(
			data.FlowId,
			user,
			&flowJson,
			&lineChannel,
			macros,
			cl)

		helpers.Log(logrus.DebugLevel, "processing action: "+action)

		callerId := event.Args[2]
		sipCallId := event.Args[3]
		fmt.Printf("Starting stasis with extension: %s, caller id: %s SIP call id: %s", exten, callerId, sipCallId)
		go processIncomingCall(cl, flow, &lineChannel, exten, callerId, sipCallId)
	case "OUTGOING_PROXY_ENDPOINT":

		callerId := event.Args[2]
		domain := event.Args[3]

		lineChannel := types.NewChannel(h, true)

		helpers.Log(logrus.DebugLevel, "looking up domain: "+domain)
		resp, err := api.GetUserByDomain(domain)

		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not get domain. error: "+err.Error())
			return
		}
		helpers.Log(logrus.DebugLevel, "workspace ID= "+strconv.Itoa(resp.WorkspaceId))
		user := types.NewUser(resp.Id, resp.WorkspaceId, resp.WorkspaceName)

		fmt.Printf("Received call from %s, domain: %s\r\n", callerId, domain)
		fmt.Printf("Calling %s\r\n", exten)
		lineChannel.Answer()
		err = utils.StartOutboundCall(cl, lineChannel.Channel.Key(), user, &lineChannel, callerId, exten, "extension", nil)
		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not create bridge. error: "+err.Error())
			lineChannel.SafeHangup()
			return

		}

	case "OUTGOING_PROXY":
		callerId := event.Args[2]
		domain := event.Args[3]

		helpers.Log(logrus.DebugLevel, "channel key: "+h.Key().ID)

		lineChannel := types.NewChannel(h, true)
		resp, err := api.GetUserByDomain(domain)

		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not get domain. error: "+err.Error())
			return
		}
		helpers.Log(logrus.DebugLevel, "workspace ID= "+strconv.Itoa(resp.WorkspaceId))
		user := types.NewUser(resp.Id, resp.WorkspaceId, resp.WorkspaceName)

		fmt.Printf("Received call from %s, domain: %s\r\n", callerId, domain)

		callerInfo, err := api.GetCallerId(user.Workspace.Domain, callerId)

		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not get caller id. error: "+err.Error())
			return
		}
		fmt.Printf("setup caller id: " + callerInfo.CallerId)
		lineChannel.Answer()
		err = utils.StartOutboundCall(cl, lineChannel.Channel.Key(), user, &lineChannel, callerInfo.CallerId, exten, "pstn", nil)
		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not create bridge. error: "+err.Error())
			lineChannel.SafeHangup()
			return

		}

	case "OUTGOING_PROXY_MEDIA":
		helpers.Log(logrus.InfoLevel, "media service call..")
	case "OUTGOING_TRUNK_CALL":
		callerId := event.Args[2]
		trunkSourceIp := event.Args[3]
		helpers.Log(logrus.DebugLevel, "channel key: "+h.Key().ID)

		lineChannel := types.NewChannel(h, true)
		resp, err := api.GetUserByTrunkSourceIp(trunkSourceIp)

		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not get domain. error: "+err.Error())
			return
		}
		helpers.Log(logrus.DebugLevel, "workspace ID= "+strconv.Itoa(resp.WorkspaceId))
		user := types.NewUser(resp.Id, resp.WorkspaceId, resp.WorkspaceName)

		fmt.Printf("Received call from %s, domain: %s\r\n", callerId, resp.WorkspaceName)
		fmt.Printf("setup caller id: " + callerId)
		lineChannel.Answer()
		headers := make([]string, 0)
		headers = append(headers, "X-Lineblocs-User-SIP-Trunk-Calling-PSTN: true")
		err = utils.StartOutboundCall(cl, lineChannel.Channel.Key(), user, &lineChannel, callerId, exten, "pstn", &headers)
		if err != nil {
			helpers.Log(logrus.DebugLevel, "could not create bridge. error: "+err.Error())
			lineChannel.SafeHangup()
			return

		}

	default:
		helpers.Log(logrus.InfoLevel, "unknown call type...")
	}
}
