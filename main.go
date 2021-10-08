package main

import (
	    "github.com/joho/godotenv"
	"os"
		"context"
	"sync"
	"fmt"
	"bytes"
	"time"
	"strconv"
	"io/ioutil"
	"net/http"
	"net/url"
	"errors"
	"encoding/json"

	"github.com/inconshreveable/log15"
	"github.com/rotisserie/eris"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/client/native"
	"github.com/CyCoreSystems/ari/v5/ext/play"
	"github.com/CyCoreSystems/ari/v5/rid"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/logger"
	"lineblocs.com/processor/mngrs"
)

var ariApp = "lineblocs"

var bridge *ari.BridgeHandle
var log log15.Logger

type APIResponse struct {
	Headers http.Header
	Body []byte
}

func logFormattedMsg(msg string) {
	log.Debug(fmt.Sprintf("msg = %s", msg))

}
func sendHttpRequest(path string, payload []byte) (*APIResponse, error) {
    url := "https://internals.lineblocs.com" + path
    fmt.Println("URL:>", url)

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
    req.Header.Set("X-Custom-Header", "myvalue")
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
		return nil, err
    }
    defer resp.Body.Close()

	var headers http.Header




    fmt.Println("response Status:", resp.Status)
    fmt.Println("response Headers:", resp.Header)

	headers = resp.Header
    body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	bodyAsString := string(body)
    fmt.Println("response Body:", bodyAsString)
    fmt.Println("response Status:", resp.Status)

status := resp.StatusCode
	if !(status >= 200 && status <= 299) {
		return nil, errors.New("Status: " + resp.Status + " result: " + bodyAsString)
	}

	return &APIResponse{  
		Headers: headers,
		Body: body }, nil

}


func sendPutRequest(path string, payload []byte) (string, error) {
    url := "https://internals.lineblocs.com" + path
    fmt.Println("URL:>", url)

    req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
		return "", err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	bodyAsString := string(body)
    fmt.Println("response Body:", bodyAsString)
    fmt.Println("response Status:", resp.Status)
status := resp.StatusCode
	if !(status >= 200 && status <= 299) {
		return "", errors.New("Status: " + resp.Status + " result: " + bodyAsString)
	}
	return bodyAsString, nil

}

func sendGetRequest(path string, vals map[string] string) (string, error) {
    fullUrl := "https://internals.lineblocs.com" + path + "?"

	for k,v := range vals {
		fullUrl = fullUrl + (k + "=" + url.QueryEscape(v))
	}
    fmt.Println("URL:>", fullUrl)

    req, err := http.NewRequest("GET", fullUrl, bytes.NewBuffer([]byte("")))
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
		return "", err
    }
    defer resp.Body.Close()
    body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	bodyAsString := string(body)

    fmt.Println("response Body:", bodyAsString)
    fmt.Println("response Status:", resp.Status)
	status := resp.StatusCode
	if !(status >= 200 && status <= 299) {
		return "", errors.New("Status: " + resp.Status + " result: " + bodyAsString)
	}
	return bodyAsString, nil
}

func createARIConnection(connectCtx context.Context) (ari.Client, error) {
	cl, err := native.Connect(&native.Options{
		Application:  ariApp,
		Username:     os.Getenv("ARI_USERNAME"),
		Password:     os.Getenv("ARI_PASSWORD"),
		URL:          os.Getenv("ARI_URL"),
		WebsocketURL: os.Getenv("ARI_WSURL") })
	if err != nil {
		log.Error("Failed to build native ARI client", "error", err)
		return nil, err
	}
	return cl, err
}
func main() {
 	log = log15.New()
	// OPTIONAL: setup logging
	native.Logger = log

	log.Info("Connecting")
	 err := godotenv.Load()
	if err != nil {
		log.Info("Error loading .env file")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	connectCtx, cancel2 := context.WithCancel(context.Background())
	defer cancel()
	defer cancel2()
	cl, err := createARIConnection(connectCtx)
	log.Info("Connected to ARI")

	defer cl.Close()
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

				_, err = sendHttpRequest( "/call/updateCall", body)
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
func ensureBridge(ctx context.Context, cl ari.Client, src *ari.Key) (err error) {
	if bridge != nil {
		log.Debug("Bridge already exists")
		return nil
	}

	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
	bridge, err = cl.Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		bridge = nil
		return eris.Wrap(err, "failed to create bridge")
	}

	wg := new(sync.WaitGroup)
	wg.Add(1)
	go manageBridge(ctx, bridge, wg)
	wg.Wait()

	return nil
}

func manageBridge(ctx context.Context, h *ari.BridgeHandle, wg *sync.WaitGroup) {
	// Delete the bridge when we exit
	defer h.Delete()

	destroySub := h.Subscribe(ari.Events.BridgeDestroyed)
	defer destroySub.Cancel()

	enterSub := h.Subscribe(ari.Events.ChannelEnteredBridge)
	defer enterSub.Cancel()

	leaveSub := h.Subscribe(ari.Events.ChannelLeftBridge)
	defer leaveSub.Cancel()

	wg.Done()
	for {
		select {
		case <-ctx.Done():
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
			go func() {
				log.Debug("Playing sound...")
				if err := play.Play(ctx, h, play.URI("sound:hello-world")).Err(); err != nil {
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
				if err := play.Play(ctx, h, play.URI("sound:confbridge-leave")).Err(); err != nil {
					log.Error("failed to play leave sound", "error", err)
				}
			}()
		}
	}
}

type Instruction func( context *types.Context, flow *types.Flow)

func startProcessingFlow( cl ari.Client, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, eventVars map[string] string, cell *types.Cell, runner *types.Runner) {
	log.Debug("processing cell type " + cell.Cell.Type)
	if runner.Cancelled {
		log.Debug("flow runner was cancelled - exiting")
		return
	}
	log.Debug("source link count: " + strconv.Itoa( len( cell.SourceLinks )))
	log.Debug("target link count: " + strconv.Itoa( len( cell.TargetLinks )))
	lineCtx := types.NewContext(
		cl,
		ctx,
		&log,
		flow,
		cell,
		runner,
		lineChannel)
	// execute it
	switch ; cell.Cell.Type {
		case "devs.LaunchModel":
			for _, link := range cell.SourceLinks {
				go startProcessingFlow( cl, ctx, flow, lineChannel, eventVars, link.Target, runner)
			}
		case "devs.SwitchModel":
		case "devs.BridgeModel":
			mngr := mngrs.NewBridgeManager(lineCtx, flow)
			mngr.StartProcessing()
		case "devs.DialModel":
		default:
	}
}
func processFlow( cl ari.Client, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, eventVars map[string] string, cell *types.Cell) {
	log.Debug("processing cell type " + cell.Cell.Type)
	runner:=types.Runner{Cancelled: false}
	flow.Runners = append( flow.Runners, &runner )
	startProcessingFlow( cl, ctx, flow, lineChannel, eventVars, cell, &runner)
}
func processIncomingCall(cl ari.Client, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, exten string, callerId string ) {
	go attachDTMFListeners( lineChannel, ctx )
	callChannel := make(chan *types.Call)
	go attachChannelLifeCycleListeners( flow, lineChannel, ctx, callChannel )

	log.Debug("calling API to create call...")
	params := types.CallParams{
		From: exten,
		To: callerId,
		Status: "start",
		Direction: "inbound",
		UserId:  flow.User.Id,
		WorkspaceId: flow.User.Workspace.Id }
	body, err := json.Marshal( params )
	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}


	log.Info("creating call...")
	resp, err := sendHttpRequest( "/call/createCall", body)

	id := resp.Headers.Get("x-call-id")
	log.Debug("Call ID is: " + id)
	idAsInt, err := strconv.Atoi(id)
	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return
	}

	call := types.Call{
		CallId: idAsInt,
		Channel: lineChannel,
		Started: time.Now(),
		Params: &params }

		flow.RootCall = &call
	log.Debug("answering call..")
	lineChannel.Channel.Answer()
	vars := make( map[string] string )
	go processFlow( cl, ctx, flow, lineChannel, vars, flow.Cells[ 0 ])
	callChannel <-  &call
	for {
		select {
			case <-ctx.Done():
				return
		}
	}
}
func startExecution(cl ari.Client, event *ari.StasisStart, ctx context.Context,  h *ari.ChannelHandle) {
	log.Info("running app", "channel", h.Key().ID)

	action := event.Args[ 0 ]
	exten := event.Args[ 1 ]
	vals := make(map[string] string)
	vals["number"] = exten

	if action == "h" { // dont handle it
		fmt.Println("Received h handler - not processing")
		return
	} else if action == "DID_DIAL" {
		fmt.Println("Already dialed - not processing")
		return
	}

	body, err := sendGetRequest("/user/getDIDNumberData", vals)

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

	body, err = sendGetRequest("/user/getWorkspaceMacros", vals)

	if err != nil {
		log.Error("startExecution err " + err.Error())
		return
	}
	var macros []types.WorkspaceMacro
 	err = json.Unmarshal( []byte(body), &macros)
	if err != nil {
		log.Error("startExecution err " + err.Error())
		return
	}


	lineChannel := types.LineChannel{
		Channel: h }
	user := types.User{
		Workspace: types.Workspace{
			Id: data.WorkspaceId },
		Id: data.CreatorId }
	flow := types.NewFlow(
		&user,
		&flowJson,
		&lineChannel, 
		cl)


		log.Debug("processing action: " + action)



	if action == "INCOMING_CALL" {
		callerId := event.Args[ 2 ]
		fmt.Printf("Starting stasis with extension: %s, caller id: %s", exten, callerId)
		go processIncomingCall( cl, ctx, flow, &lineChannel, exten, callerId )
	} else if action == "OUTGOING_PROXY_ENDPOINT" {

	} else if action == "OUTGOING_PROXY" {

	} else if action == "OUTGOING_PROXY_MEDIA" {

	}
	/*
	if err := h.Answer(); err != nil {
		log.Error("failed to answer call", "error", err)
		// return
	}

	if err := ensureBridge(ctx, cl, h.Key()); err != nil {
		log.Error("failed to manage bridge", "error", err)
		return
	}

	if err := bridge.AddChannel(h.Key().ID); err != nil {
		log.Error("failed to add channel to bridge", "error", err)
		return
	}

	log.Info("channel added to bridge")
	*/
	return
}