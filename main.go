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
	helpers.Log(logrus.InfoLevel, "Connecting")
	connectCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cl, err := createARIConnection(connectCtx)

	if err != nil {
		fmt.Printf("could not connect to ARI. error: %s", err.Error())
		panic(err.Error())
		return
	}

	helpers.Log(logrus.InfoLevel, "Connected to ARI")

	defer cl.Close()

	helpers.Log(logrus.InfoLevel, "starting GRPC listener...")
	go grpc.StartListener(cl)
	// setup app

	helpers.Log(logrus.InfoLevel, "Starting listener app")

	helpers.Log(logrus.InfoLevel, "Listening for new calls")
	sub := cl.Bus().Subscribe(nil, "StasisStart")

	for {
		if !cl.Connected() {
			helpers.Log(logrus.ErrorLevel, "websocket was disconnected. reconnecting now.")
			cl.Close()
			startProcessingWSEvents()
			return
		}

		select {
			case e := <-sub.Events():
				v := e.(*ari.StasisStart)
				helpers.Log(logrus.InfoLevel, "Got stasis start"+" channel "+v.Channel.ID)
				go startExecution(cl, v, cl.Channel().Get(v.Key(ari.ChannelKey, v.Channel.ID)))
			case <-connectCtx.Done():
				cl.Close()
				return
			}
	}
}

func main() {
	// OPTIONAL: setup logging
	//native.Logger = log
	// Init Logrus and configure channels
	logDestination := utils.Config("LOG_DESTINATIONS")
	helpers.InitLogrus(logDestination)

	startProcessingWSEvents()
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
	var call *types.Call
	endSub := channel.Channel.Subscribe(ari.Events.StasisEnd)
	defer endSub.Cancel()

	call = nil

	for {

		select {
		case <-endSub.Events():
			helpers.Log(logrus.DebugLevel, "stasis end called..")
			call.Ended = time.Now()
			params := types.StatusParams{
				CallId: call.CallId,
				Ip:     utils.GetPublicIp(),
				Status: "ended"}
			body, err := json.Marshal(params)
			if err != nil {
				helpers.Log(logrus.DebugLevel, "JSON error: "+err.Error())
				continue
			}

			_, err = api.SendHttpRequest("/call/updateCall", body)
			if err != nil {
				helpers.Log(logrus.DebugLevel, "HTTP error: "+err.Error())
				continue
			}
			err = createCallDebit(flow.User, call, "incoming")
			if err != nil {
				helpers.Log(logrus.DebugLevel, "HTTP error: "+err.Error())
				continue
			}

		case call = <-callChannel:
			helpers.Log(logrus.DebugLevel, "call is setup")
			helpers.Log(logrus.DebugLevel, "id is "+strconv.Itoa(call.CallId))
		}
	}
}
func attachDTMFListeners(channel *types.LineChannel) {
	dtmfSub := channel.Channel.Subscribe(ari.Events.ChannelDtmfReceived)
	defer dtmfSub.Cancel()

	for {

		select {
		case <-dtmfSub.Events():
			helpers.Log(logrus.DebugLevel, "received DTMF!")
		}
	}
}

func processIncomingCall(cl ari.Client, flow *types.Flow, lineChannel *types.LineChannel, exten string, callerId string, sipCallId string) {
	go attachDTMFListeners(lineChannel)
	callChannel := make(chan *types.Call)
	go attachChannelLifeCycleListeners(flow, lineChannel, callChannel)

	helpers.Log(logrus.DebugLevel, "calling API to create call...")
	helpers.Log(logrus.DebugLevel, "exten is: "+exten)
	helpers.Log(logrus.DebugLevel, "caller ID is: "+callerId)
	helpers.Log(logrus.DebugLevel, "SIP call id: "+sipCallId)
	params := types.CallParams{
		From:        callerId,
		To:          exten,
		Status:      "start",
		Direction:   "inbound",
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
	//ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()
	helpers.Log(logrus.InfoLevel, "running app"+" channel "+h.Key().ID)

	action := event.Args[0]
	exten := event.Args[1]
	vals := make(map[string]string)
	vals["number"] = exten

	helpers.Log(logrus.DebugLevel, "received action: "+action)
	helpers.Log(logrus.DebugLevel, "EXTEN: "+exten)

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
			return

		}

	default:
		helpers.Log(logrus.InfoLevel, "unknown call type...")
	}
}
