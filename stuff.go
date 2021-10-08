diff --git a/main.go b/main.go
index 923d8ef..77bb551 100644
--- a/main.go
+++ b/main.go
@@ -2,16 +2,12 @@ package main
 
 import (
 	    "github.com/joho/godotenv"
-	"os"
 		"context"
 	"sync"
 	"fmt"
-	"bytes"
 	"time"
 	"strconv"
-	"io/ioutil"
 	"net/http"
-	"net/url"
 	"errors"
 	"encoding/json"
 
@@ -20,12 +16,12 @@ import (
 
 	"github.com/CyCoreSystems/ari/v5"
 	"github.com/CyCoreSystems/ari/v5/client/native"
-	"github.com/CyCoreSystems/ari/v5/ext/play"
 	"github.com/CyCoreSystems/ari/v5/rid"
 	"lineblocs.com/processor/types"
 	"lineblocs.com/processor/utils"
 	"lineblocs.com/processor/logger"
 	"lineblocs.com/processor/mngrs"
+	"lineblocs.com/processor/api"
 )
 
 var ariApp = "lineblocs"
@@ -42,124 +38,116 @@ func logFormattedMsg(msg string) {
 	log.Debug(fmt.Sprintf("msg = %s", msg))
 
 }
-func sendHttpRequest(path string, payload []byte) (*APIResponse, error) {
-    url := "https://internals.lineblocs.com" + path
-    fmt.Println("URL:>", url)
 
-    req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
-    req.Header.Set("X-Custom-Header", "myvalue")
-    req.Header.Set("Content-Type", "application/json")
+func manageBridge(bridge *types.LineBridge, wg *sync.WaitGroup) {
+	h := bridge.Bridge
 
-    client := &http.Client{}
-    resp, err := client.Do(req)
-    if err != nil {
-		return nil, err
-    }
-    defer resp.Body.Close()
+	log.Debug("manageBridge called..")
+	// Delete the bridge when we exit
+	defer h.Delete()
 
-	var headers http.Header
+	destroySub := h.Subscribe(ari.Events.BridgeDestroyed)
+	defer destroySub.Cancel()
 
+	enterSub := h.Subscribe(ari.Events.ChannelEnteredBridge)
+	defer enterSub.Cancel()
 
+	leaveSub := h.Subscribe(ari.Events.ChannelLeftBridge)
+	defer leaveSub.Cancel()
 
+	wg.Done()
+	log.Debug("listening for bridge events...")
+	for {
+		select {
+		case <-destroySub.Events():
+			log.Debug("bridge destroyed")
+			return
+		case e, ok := <-enterSub.Events():
+			if !ok {
+				log.Error("channel entered subscription closed")
+				return
+			}
+			v := e.(*ari.ChannelEnteredBridge)
+			log.Debug("channel entered bridge", "channel", v.Channel.Name)
+			//go man.startOutboundCall(bridge, wg) 
+			
+			func() {
+				log.Debug("starting bridge....")
+			}()
+		case e, ok := <-leaveSub.Events():
+			if !ok {
+				log.Error("channel left subscription closed")
+				return
+			}
+			v := e.(*ari.ChannelLeftBridge)
+			log.Debug("channel left bridge", "channel", v.Channel.Name)
+			go func() {
+			}()
+		}
+	}
+}
 
-    fmt.Println("response Status:", resp.Status)
-    fmt.Println("response Headers:", resp.Header)
+func ensureBridge( cl ari.Client,	src *ari.Key, user *types.User, lineChannel *types.LineChannel, callerId string, numberToCall string	) (error) {
+	log.Debug("ensureBridge called..")
+	var bridge *ari.BridgeHandle 
+	var err error
 
-	headers = resp.Header
-    body, err := ioutil.ReadAll(resp.Body)
+	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
+	bridge, err = cl.Bridge().Create(key, "mixing", key.ID)
 	if err != nil {
-		return nil, err
+		bridge = nil
+		return eris.Wrap(err, "failed to create bridge")
 	}
 
-	bodyAsString := string(body)
-    fmt.Println("response Body:", bodyAsString)
-    fmt.Println("response Status:", resp.Status)
+	lineBridge := types.LineBridge{Bridge: bridge}
+	log.Info("channel added to bridge")
 
-status := resp.StatusCode
-	if !(status >= 200 && status <= 299) {
-		return nil, errors.New("Status: " + resp.Status + " result: " + bodyAsString)
+	wg := new(sync.WaitGroup)
+	wg.Add(1)
+	go manageBridge(&lineBridge, wg)
+	wg.Wait()
+	if err := bridge.AddChannel(lineChannel.Channel.Key().ID); err != nil {
+		log.Error("failed to add channel to bridge", "error", err)
+		return errors.New( "failed to add channel to bridge" )
 	}
 
-	return &APIResponse{  
-		Headers: headers,
-		Body: body }, nil
-
-}
-
-
-func sendPutRequest(path string, payload []byte) (string, error) {
-    url := "https://internals.lineblocs.com" + path
-    fmt.Println("URL:>", url)
+	log.Info("channel added to bridge")
 
-    req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
-    req.Header.Set("Content-Type", "application/json")
+	// create outbound leg
+	outboundChannel, err := cl.Channel().Create(nil, utils.CreateChannelRequest2( numberToCall )	)
 
-    client := &http.Client{}
-    resp, err := client.Do(req)
-    if err != nil {
-		return "", err
-    }
-    defer resp.Body.Close()
-    body, err := ioutil.ReadAll(resp.Body)
 	if err != nil {
-		return "", err
+		log.Debug("error creating outbound channel: " + err.Error())
+		return err
 	}
-	bodyAsString := string(body)
-    fmt.Println("response Body:", bodyAsString)
-    fmt.Println("response Status:", resp.Status)
-status := resp.StatusCode
-	if !(status >= 200 && status <= 299) {
-		return "", errors.New("Status: " + resp.Status + " result: " + bodyAsString)
-	}
-	return bodyAsString, nil
-
-}
-
-func sendGetRequest(path string, vals map[string] string) (string, error) {
-    fullUrl := "https://internals.lineblocs.com" + path + "?"
 
-	for k,v := range vals {
-		fullUrl = fullUrl + (k + "=" + url.QueryEscape(v))
-	}
-    fmt.Println("URL:>", fullUrl)
-
-    req, err := http.NewRequest("GET", fullUrl, bytes.NewBuffer([]byte("")))
-    req.Header.Set("Content-Type", "application/json")
-
-    client := &http.Client{}
-    resp, err := client.Do(req)
-    if err != nil {
-		return "", err
-    }
-    defer resp.Body.Close()
-    body, err := ioutil.ReadAll(resp.Body)
+	params := types.CallParams{
+		From: callerId,
+		To: numberToCall,
+		Status: "start",
+		Direction: "outbound",
+		UserId:  user.Id,
+		WorkspaceId: user.Workspace.Id }
+	body, err := json.Marshal( params )
 	if err != nil {
-		return "", err
+		log.Error( "error occured: " + err.Error() )
+		return err
 	}
-	bodyAsString := string(body)
 
-    fmt.Println("response Body:", bodyAsString)
-    fmt.Println("response Status:", resp.Status)
-	status := resp.StatusCode
-	if !(status >= 200 && status <= 299) {
-		return "", errors.New("Status: " + resp.Status + " result: " + bodyAsString)
-	}
-	return bodyAsString, nil
-}
+	log.Info("creating outbound call...")
+	resp, err := api.SendHttpRequest( "/call/createCall", body )
+	outChannel := types.LineChannel{
+		Channel: outboundChannel }
+	_, err = utils.CreateCall( resp.Headers.Get("x-call-id"), &outChannel, &params)
 
-func createARIConnection(connectCtx context.Context) (ari.Client, error) {
-	cl, err := native.Connect(&native.Options{
-		Application:  ariApp,
-		Username:     os.Getenv("ARI_USERNAME"),
-		Password:     os.Getenv("ARI_PASSWORD"),
-		URL:          os.Getenv("ARI_URL"),
-		WebsocketURL: os.Getenv("ARI_WSURL") })
 	if err != nil {
-		log.Error("Failed to build native ARI client", "error", err)
-		return nil, err
+		log.Error( "error occured: " + err.Error() )
+		return err
 	}
-	return cl, err
+	return nil
 }
+
+
 func main() {
  	log = log15.New()
 	// OPTIONAL: setup logging
@@ -236,7 +224,7 @@ func attachChannelLifeCycleListeners( flow* types.Flow, channel* types.LineChann
 					continue
 				}
 
-				_, err = sendHttpRequest( "/call/updateCall", body)
+				_, err = api.SendHttpRequest( "/call/updateCall", body)
 				if err != nil {
 					log.Debug("HTTP error: " + err.Error())
 					continue
@@ -268,76 +256,7 @@ func attachDTMFListeners( channel* types.LineChannel, ctx context.Context) {
 		}
 	}
 }
-func ensureBridge(ctx context.Context, cl ari.Client, src *ari.Key) (err error) {
-	if bridge != nil {
-		log.Debug("Bridge already exists")
-		return nil
-	}
 
-	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
-	bridge, err = cl.Bridge().Create(key, "mixing", key.ID)
-	if err != nil {
-		bridge = nil
-		return eris.Wrap(err, "failed to create bridge")
-	}
-
-	wg := new(sync.WaitGroup)
-	wg.Add(1)
-	go manageBridge(ctx, bridge, wg)
-	wg.Wait()
-
-	return nil
-}
-
-func manageBridge(ctx context.Context, h *ari.BridgeHandle, wg *sync.WaitGroup) {
-	// Delete the bridge when we exit
-	defer h.Delete()
-
-	destroySub := h.Subscribe(ari.Events.BridgeDestroyed)
-	defer destroySub.Cancel()
-
-	enterSub := h.Subscribe(ari.Events.ChannelEnteredBridge)
-	defer enterSub.Cancel()
-
-	leaveSub := h.Subscribe(ari.Events.ChannelLeftBridge)
-	defer leaveSub.Cancel()
-
-	wg.Done()
-	for {
-		select {
-		case <-ctx.Done():
-			return
-		case <-destroySub.Events():
-			log.Debug("bridge destroyed")
-			return
-		case e, ok := <-enterSub.Events():
-			if !ok {
-				log.Error("channel entered subscription closed")
-				return
-			}
-			v := e.(*ari.ChannelEnteredBridge)
-			log.Debug("channel entered bridge", "channel", v.Channel.Name)
-			go func() {
-				log.Debug("Playing sound...")
-				if err := play.Play(ctx, h, play.URI("sound:hello-world")).Err(); err != nil {
-					log.Error("failed to play join sound", "error", err)
-				}
-			}()
-		case e, ok := <-leaveSub.Events():
-			if !ok {
-				log.Error("channel left subscription closed")
-				return
-			}
-			v := e.(*ari.ChannelLeftBridge)
-			log.Debug("channel left bridge", "channel", v.Channel.Name)
-			go func() {
-				if err := play.Play(ctx, h, play.URI("sound:confbridge-leave")).Err(); err != nil {
-					log.Error("failed to play leave sound", "error", err)
-				}
-			}()
-		}
-	}
-}
 
 type Instruction func( context *types.Context, flow *types.Flow)
 
@@ -398,7 +317,7 @@ func processIncomingCall(cl ari.Client, ctx context.Context, flow *types.Flow, l
 
 
 	log.Info("creating call...")
-	resp, err := sendHttpRequest( "/call/createCall", body)
+	resp, err := api.SendHttpRequest( "/call/createCall", body)
 
 	id := resp.Headers.Get("x-call-id")
 	log.Debug("Call ID is: " + id)
@@ -441,72 +360,93 @@ func startExecution(cl ari.Client, event *ari.StasisStart, ctx context.Context,
 	} else if action == "DID_DIAL" {
 		fmt.Println("Already dialed - not processing")
 		return
-	}
-
-	body, err := sendGetRequest("/user/getDIDNumberData", vals)
-
-	if err != nil {
-		log.Error("startExecution err " + err.Error())
-		return
-	}
-
-	var data types.FlowDIDData
-	var flowJson types.FlowVars
- 	err = json.Unmarshal( []byte(body), &data )
-	if err != nil {
-		log.Error("startExecution err " + err.Error())
+	} else if action == "DID_DIAL_2" {
+		fmt.Println("Already dialed - not processing")
 		return
-	}
+	} else if action == "INCOMING_CALL" {
+		body, err := api.SendGetRequest("/user/getDIDNumberData", vals)
 
-	if utils.CheckFreeTrial( data.Plan ) {
-		log.Error("Ending call due to free trial")
-		h.Hangup()
-		logFormattedMsg(logger.FREE_TRIAL_ENDED)
-		return
-	}
- 	err = json.Unmarshal( []byte(data.FlowJson), &flowJson )
-	if err != nil {
-		log.Error("startExecution err " + err.Error())
-		return
-	}
+		if err != nil {
+			log.Error("startExecution err " + err.Error())
+			return
+		}
 
-	body, err = sendGetRequest("/user/getWorkspaceMacros", vals)
+		var data types.FlowDIDData
+		var flowJson types.FlowVars
+		err = json.Unmarshal( []byte(body), &data )
+		if err != nil {
+			log.Error("startExecution err " + err.Error())
+			return
+		}
 
-	if err != nil {
-		log.Error("startExecution err " + err.Error())
-		return
-	}
-	var macros []types.WorkspaceMacro
- 	err = json.Unmarshal( []byte(body), &macros)
-	if err != nil {
-		log.Error("startExecution err " + err.Error())
-		return
-	}
+		if utils.CheckFreeTrial( data.Plan ) {
+			log.Error("Ending call due to free trial")
+			h.Hangup()
+			logFormattedMsg(logger.FREE_TRIAL_ENDED)
+			return
+		}
+		err = json.Unmarshal( []byte(data.FlowJson), &flowJson )
+		if err != nil {
+			log.Error("startExecution err " + err.Error())
+			return
+		}
 
+		body, err = api.SendGetRequest("/user/getWorkspaceMacros", vals)
 
-	lineChannel := types.LineChannel{
-		Channel: h }
-	user := types.User{
-		Workspace: types.Workspace{
-			Id: data.WorkspaceId },
-		Id: data.CreatorId }
-	flow := types.NewFlow(
-		&user,
-		&flowJson,
-		&lineChannel, 
-		cl)
+		if err != nil {
+			log.Error("startExecution err " + err.Error())
+			return
+		}
+		var macros []types.WorkspaceMacro
+		err = json.Unmarshal( []byte(body), &macros)
+		if err != nil {
+			log.Error("startExecution err " + err.Error())
+			return
+		}
 
 
-		log.Debug("processing action: " + action)
+		lineChannel := types.LineChannel{
+			Channel: h }
+		user := types.User{
+			Workspace: types.Workspace{
+				Id: data.WorkspaceId },
+			Id: data.CreatorId }
+		flow := types.NewFlow(
+			&user,
+			&flowJson,
+			&lineChannel, 
+			cl)
 
 
+		log.Debug("processing action: " + action)
 
-	if action == "INCOMING_CALL" {
 		callerId := event.Args[ 2 ]
 		fmt.Printf("Starting stasis with extension: %s, caller id: %s", exten, callerId)
 		go processIncomingCall( cl, ctx, flow, &lineChannel, exten, callerId )
 	} else if action == "OUTGOING_PROXY_ENDPOINT" {
 
+		callerId := event.Args[ 2 ]
+		domain := event.Args[ 3 ]
+
+
+		lineChannel := types.LineChannel{
+			Channel: h }
+
+		resp, err := api.GetUserByDomain( domain )
+		user := types.User{
+			Workspace: types.Workspace{
+				Id: resp.WorkspaceId },
+			Id: resp.Id  }
+
+		fmt.Printf("Received call from %s, domain: %s\r\n", callerId, domain)
+			ensureBridge(
+				cl,
+				lineChannel.Channel.Key(),
+				&user,
+				&lineChannel,
+			exten,
+		callerId)
+
 	} else if action == "OUTGOING_PROXY" {
 
 	} else if action == "OUTGOING_PROXY_MEDIA" {
