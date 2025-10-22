package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	speech "cloud.google.com/go/speech/apiv1"
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/ext/record"
	"github.com/CyCoreSystems/ari/v5/rid"
	helpers "github.com/Lineblocs/go-helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"
	"github.com/ivahaev/amigo"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"lineblocs.com/processor/api"
	"lineblocs.com/processor/types"
	processor_helpers "lineblocs.com/processor/helpers"
)

var log *logrus.Logger

type ConfCache struct {
	Id       string          `json:"id"`
	BridgeId string          `json:"bridgeId"`
	UserInfo *types.UserInfo `json:"userInfo"`
}

// TODO get the ip
func GetPublicIp() string {
	return "0.0.0.0"
}

func PlaybackLoops(data types.ModelData) int {
	item, ok := data.(types.ModelDataStr)

	if !ok {
		return 1
	}
	if item.Value == "" {
		// default caller id
		return 1
	}
	intVar, err := strconv.Atoi(item.Value)
	if err != nil {
		return 1
	}
	return intVar
}
func DetermineCallerId(call *types.Call, data types.ModelData) string {
	item, ok := data.(types.ModelDataStr)

	if !ok {
		return call.Params.From
	}
	if item.Value == "" {
		// default caller id
		return call.Params.From
	}
	return item.Value
}

func CheckFreeTrial(plan string) bool {
	if plan == "expired" {
		return true
	}
	return false
}

func CreateRDB() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return rdb
}

func FindLinkByName(links []*types.Link, direction string, tag string) (*types.Link, error) {
	for _, link := range links {
		if direction == "source" {
			if link.Link.Source.Port == tag {
				return link, nil
			}
		} else if direction == "target" {

			if link.Link.Target.Port == tag {
				return link, nil
			}
		}
	}
	return nil, errors.New("Could not find link")
}
func GetCellByName(flow *types.Flow, name string) (*types.Cell, error) {
	for _, v := range flow.Cells {

		if v.Cell.Name == name {
			return v, nil
		}
	}
	return nil, nil
}
func LookupCellVariable(flow *types.Flow, name string, lookup string) (string, error) {
	var cell *types.Cell
	cell, err := GetCellByName(flow, name)
	if err != nil {
		return "", err
	}
	if cell == nil {
		return "", errors.New("Could not find cell")
	}
	fmt.Println("looking up cell variable\r\n")
	fmt.Println(cell.Cell.Type)
	if cell.Cell.Type == "devs.LaunchModel" {
		if lookup == "call.from" {
			return cell.EventVars["callFrom"], nil
		} else if lookup == "call.to" {
			return cell.EventVars["callTo"], nil
		} else if lookup == "channel.id" {
			return cell.EventVars["channelId"], nil
		}
	} else if cell.Cell.Type == "devs.DialhModel" {
		if lookup == "from" {
			return cell.EventVars["from"], nil
		} else if lookup == "call.to" {
			return cell.EventVars["to"], nil
		} else if lookup == "dial_status" {
			return cell.EventVars["dial_status"], nil
		} else if lookup == "channel.id" {
			return cell.EventVars["channelId"], nil
		}
	} else if cell.Cell.Type == "devs.BridgehModel" {
		if lookup == "from" {
			return cell.EventVars["from"], nil
		} else if lookup == "call.to" {
			return cell.EventVars["to"], nil
		} else if lookup == "dial_status" {
			return cell.EventVars["dial_status"], nil
		} else if lookup == "channel.id" {
			return cell.EventVars["channelId"], nil
		} else if lookup == "started" {
			call := cell.AttachedCall
			return strconv.Itoa(call.GetStartTime()), nil
		} else if lookup == "ended" {
			call := cell.AttachedCall
			return strconv.Itoa(call.FigureOutEndedTime()), nil
		}
	} else if cell.Cell.Type == "devs.ProcessInputModel" {
		fmt.Println("getting input value..\r\n")
		if lookup == "digits" {
			fmt.Println("found:")
			fmt.Println(cell.EventVars["digits"])
			return cell.EventVars["digits"], nil
		}
	}
	return "", errors.New("Could not find link")
}

// TODO call API to get proxy IPs
func GetSIPProxy() string {
	return os.Getenv("PROXY_HOST")
}

func GetARIHost() string {
	return os.Getenv("ARI_HOST")
}

func CreateChannelRequest(numberToCall string) ari.ChannelCreateRequest {
	return ari.ChannelCreateRequest{
		Endpoint: "SIP/" + numberToCall + "@" + GetSIPProxy(),
		App:      "lineblocs",
		AppArgs:  "PROCESSED_CALL,"}
}

func CreateOriginateRequest(callerId string, numberToCall string, headers map[string]string) ari.OriginateRequest {
	return ari.OriginateRequest{
		CallerID: callerId,
		Endpoint: "SIP/" + numberToCall + "@" + GetSIPProxy(),
		App:      "lineblocs",
		AppArgs:  "PROCESSED_CALL,", Variables: headers}
}

func CreateOriginateRequest2(callerId string, numberToCall string) ari.OriginateRequest {
	return ari.OriginateRequest{
		CallerID: callerId,
		Endpoint: "SIP/" + numberToCall + "/" + GetSIPProxy(),
		App:      "lineblocs",
		AppArgs:  "PROCESSED_CALL_2,"}
}

func DetermineNumberToCall(data map[string]types.ModelData) (string, error) {
	callType, ok := data["call_type"].(types.ModelDataStr)
	if !ok {
		return "", errors.New("Could not get call type")
	}

	switch callType.Value {
	case "Extension":
		ext, ok := data["extension"].(types.ModelDataStr)
		if !ok {
			return "", errors.New("Could not get ext")
		}
		return ext.Value, nil
	case "Phone Number":
		ext, ok := data["number_to_call"].(types.ModelDataStr)
		if !ok {
			return "", errors.New("Could not get number")
		}
		return ext.Value, nil
	}
	return "", errors.New("Unknown call type")
}

func GetSIPSecretKey() string {
	//return "BrVIsXzQx9-7lvRsXMC2V57dA4UEc-G_HwnCpK-zctk"
	//return "BrVIsXzQx9-7lvRsXMC2V57dA4UEc-G_HwnCpK-zctk"
	key := os.Getenv("LINEBLOCS_KEY")
	return key
}

func CreateSIPHeaders(domain, callerId, typeOfCall, apiCallId string, addedHeaders *[]string) map[string]string {
	headers := make(map[string]string)
	headers["SIPADDHEADER0"] = "X-LineBlocs-Key: " + GetSIPSecretKey()
	headers["SIPADDHEADER1"] = "X-LineBlocs-Domain: " + domain
	headers["SIPADDHEADER2"] = "X-LineBlocs-Route-Type: " + typeOfCall
	headers["SIPADDHEADER3"] = "X-LineBlocs-Caller: " + callerId
	headers["SIPADDHEADER4"] = "X-LineBlocs-API-CallId: " + apiCallId
	headerCounter := 5
	if addedHeaders != nil {
		for _, value := range *addedHeaders {
			headers["SIPADDHEADER"+strconv.Itoa(headerCounter)] = value
			headerCounter = headerCounter + 1
		}
	}
	return headers
}

func CreateSIPHeadersForSIPTrunkCall(domain, callerId, typeOfCall, apiCallId string, trunkAddr string) map[string]string {
	headers := make(map[string]string)
	headers["SIPADDHEADER0"] = "X-LineBlocs-Key: " + GetSIPSecretKey()
	headers["SIPADDHEADER1"] = "X-LineBlocs-Domain: " + domain
	headers["SIPADDHEADER2"] = "X-LineBlocs-Route-Type: " + typeOfCall
	headers["SIPADDHEADER3"] = "X-LineBlocs-Caller: " + callerId
	headers["SIPADDHEADER4"] = "X-LineBlocs-API-CallId: " + apiCallId
	headers["SIPADDHEADER5"] = "X-Lineblocs-User-SIP-Trunk-Addr: " + trunkAddr
	headers["SIPADDHEADER6"] = "X-Lineblocs-User-SIP-Trunk: true"
	return headers
}

func CreateKafkaProducer() (*kafka.Producer, error) {
	servers := os.Getenv("KAFKA_SERVER_ENDPOINTS")
	producerEnabled := false
	clientId := "lineblocs-call-processor"
	if !producerEnabled {
		return nil, nil
	}

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": servers,
		"client.id": clientId,
		"acks": "all"})

	if err != nil {
		return nil, err
	}

	return p, nil
}

func sendToAssetServer(path string, filename string) (string, error) {
	settings, err := api.GetSettings()
	if err != nil {
		return "", err
	}

	creds := credentials.NewStaticCredentials(
		settings.Credentials["aws_access_key_id"],
		settings.Credentials["aws_secret_access_key"],
		"")

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(settings.Credentials["aws_region"]),
		Credentials: creds,
	})
	if err != nil {
		return "", fmt.Errorf("error occured: %v", err)
	}

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file %q, %v", path, err)
	}

	bucket := "lineblocs"
	key := "media-streams/" + filename

	fmt.Printf("Uploading to %s\r\n", key)
	// Upload the file to S3.
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   f,
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file, %v", err)
	}
 	
	s3Url := aws.StringValue(&result.Location)
	fmt.Printf("file uploaded to, %s\n", s3Url)

	// send back link to media
	//url := "https://mediafs." + os.Getenv("DEPLOYMENT_DOMAIN") + "/" + key
	//return url, nil
	return s3Url, nil
}

func DownloadFile(flow *types.Flow, url string) (string, error) {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var folder string = "/tmp/"
	uniq, err := uuid.NewUUID()
	if err != nil {
		helpers.Log(logrus.ErrorLevel, err.Error())
		return "", err
	}

	var filename string = url
	var ext = path.Ext(filename)
	//var name = filename[0:len(filename)-len(extension)]
	filename = (uniq.String() + "." + ext)
	filepath := folder + filename
	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	fullPathToFile, err := changeAudioEncoding(filepath, ext)
	if err != nil {
		return "", err
	}

	link, err := sendToAssetServer(fullPathToFile, filename)
	if err != nil {
		return "", err
	}

	return link, err
}

func StartTTS(say string, gender string, voice string, lang string) (string, error) {
	// Instantiates a client.
	ctx := context.Background()
	settings, err := api.GetSettings()
	if err != nil {
		return "", err
	}
	var serviceAccountKey = []byte(settings.Credentials["google_service_account_json"])

	creds, err := google.CredentialsFromJSON(ctx, serviceAccountKey, "https://www.googleapis.com/auth/cloud-platform")

	if err != nil {
		helpers.Log(logrus.ErrorLevel, err.Error())
		return "", err
	}
	ctx2 := context.Background()
	//client, err := texttospeech.NewClient(ctx)
	opt := option.WithCredentials(creds)
	client, err := texttospeech.NewClient(ctx2, opt)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, err.Error())
		return "", err
	}
	defer client.Close()

	var ssmlGender texttospeechpb.SsmlVoiceGender
	if gender == "MALE" {
		ssmlGender = texttospeechpb.SsmlVoiceGender_MALE
	} else if gender == "FEMALE" {
		ssmlGender = texttospeechpb.SsmlVoiceGender_FEMALE
	}
	// Perform the text-to-speech request on the text input with the selected
	// voice parameters and audio file type.
	req := texttospeechpb.SynthesizeSpeechRequest{
		// Set the text input to be synthesized.
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: say},
		},
		// Build the voice request, select the language code ("en-US") and the SSML
		// voice gender ("neutral").
		Voice: &texttospeechpb.VoiceSelectionParams{
			Name:         voice,
			LanguageCode: lang,
			//SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
			SsmlGender: ssmlGender,
		},
		// Select the type of audio file you want returned.
		AudioConfig: &texttospeechpb.AudioConfig{
			//AudioEncoding: texttospeechpb.AudioEncoding_MP3,
			AudioEncoding:   texttospeechpb.AudioEncoding_LINEAR16,
			SampleRateHertz: 8000,
		},
	}

	resp, err := client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, err.Error())
		return "", err
	}

	// The resp's AudioContent is binary.
	var folder string = "/tmp/"
	uniq, err := uuid.NewUUID()
	if err != nil {
		helpers.Log(logrus.ErrorLevel, err.Error())
		return "", err
	}

	filename := (uniq.String() + ".wav")
	fullPathToFile := folder + filename

	err = ioutil.WriteFile(fullPathToFile, resp.AudioContent, 0644)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, err.Error())
		return "", err
	}
	fmt.Printf("Audio content written to file: %v\n", fullPathToFile)
	link, err := sendToAssetServer(fullPathToFile, filename)
	fmt.Printf("asset link: " + link)
	if err != nil {
		return "", err
	}

	return link, nil
}

func StartSTT(fileURI string) (string, error) {
	ctx := context.Background()
	settings, err := api.GetSettings()
	if err != nil {
		return "", err
	}
	var serviceAccountKey = []byte(settings.Credentials["google_service_account_json"])

	creds, err := google.CredentialsFromJSON(ctx, serviceAccountKey)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, err.Error())
		return "", err
	}
	ctx2 := context.Background()
	//client, err := texttospeech.NewClient(ctx)
	opt := option.WithCredentials(creds)

	// Creates a client.
	client, err := speech.NewClient(ctx2, opt)
	if err != nil {
		fmt.Printf("Failed to create client: %v", err)
		return "", err
	}
	defer client.Close()

	// Detects speech in the audio file.
	resp, err := client.Recognize(ctx, &speechpb.RecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_LINEAR16,
			SampleRateHertz: 8000,
			LanguageCode:    "en-US",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: fileURI},
		},
	})
	if err != nil {
		fmt.Printf("failed to recognize: %v", err)
		return "", err
	}

	// Prints the results.
	text := ""
	var highestConfidence float32 = 0.0
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			fmt.Printf("\"%v\" (confidence=%3f)\n", alt.Transcript, alt.Confidence)
			if highestConfidence == 0.0 || alt.Confidence > highestConfidence {
				text = alt.Transcript
				highestConfidence = alt.Confidence
			}
		}
	}
	return text, nil
}

func SaveLiveRecording(result *record.Result) (string, error) {
	var folder string = "/tmp/"
	uniq, err := uuid.NewUUID()
	if err != nil {
		helpers.Log(logrus.ErrorLevel, err.Error())
		return "", err
	}

	data := []byte("")
	filename := (uniq.String() + ".wav")
	fullPathToFile := folder + filename

	err = ioutil.WriteFile(fullPathToFile, data, 0644)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, err.Error())
		return "", err
	}
	fmt.Printf("Audio content written to file: %v\n", fullPathToFile)
	link, err := sendToAssetServer(fullPathToFile, filename)
	if err != nil {
		return "", err
	}
	return link, nil
}

func changeAudioEncoding(filepath string, ext string) (string, error) {
	newfile := filepath + ".wav"

	err := ffmpeg_go.Input(filepath).Output(newfile, ffmpeg_go.KwArgs{
		"acodec": "pcm_u8",
		"ar":     "8000",
	}).OverWriteOutput().Run()

	if err != nil {
		return "", err
	}
	return newfile, nil

}

func ParseRingTimeout(value types.ModelData) int {

	item, ok := value.(types.ModelDataStr)
	if !ok {
		return 30
	}
	result, err := strconv.Atoi(item.Value)

	// use default
	if err != nil {
		return 30
	}
	return result

}

func SafeSendResonseToChannel(channel chan<- *types.ManagerResponse, resp *types.ManagerResponse) {
}

func GetWorkspaceNameFromDomain(domain string) string {
	s := strings.Split(domain, ".")
	return s[0]
}

func AddConfBridge(client ari.Client, workspace string, confName string, conf *types.LineConference) (*types.LineConference, error) {
	var ctx = context.Background()
	key := workspace + "_" + confName
	rdb := CreateRDB()
	params := ConfCache{
		Id:       conf.Id,
		UserInfo: &conf.User.Info,
		BridgeId: conf.Bridge.Bridge.ID()}
	body, err := json.Marshal(params)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return nil, err
	}

	err = rdb.Set(ctx, key, body, 0).Err()
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func GetConfBridge(client ari.Client, user *types.User, confName string) (*types.LineConference, error) {
	var ctx = context.Background()
	key := strconv.Itoa(user.Workspace.Id) + "_" + confName
	rdb := CreateRDB()
	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	fmt.Println("key", val)
	var data ConfCache
	err = json.Unmarshal([]byte(val), &data)
	if err != nil {
		return nil, err
	}
	src := ari.NewKey(ari.BridgeKey, data.BridgeId)
	bridge := client.Bridge().Get(src)
	conf := types.NewConference(data.Id, user, &types.LineBridge{Bridge: bridge})
	return conf, nil
}

func StartOutboundCall(cl ari.Client, src *ari.Key, user *types.User, lineChannel *types.LineChannel, callerId string, numberToCall string, typeOfCall string, addedHeaders *[]string) error {
	helpers.Log(logrus.DebugLevel, "StartOutboundCall called..")
	var bridge *ari.BridgeHandle
	var err error

	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
	bridge, err = cl.Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		bridge = nil
		return eris.Wrap(err, "failed to create bridge")
	}
	outChannel := types.LineChannel{}
	lineBridge := types.NewBridge(bridge, false)

	helpers.Log(logrus.InfoLevel, "channel added to bridge")
	outboundChannel, err := cl.Channel().Create(nil, CreateChannelRequest(numberToCall))

	if err != nil {
		helpers.Log(logrus.DebugLevel, "error creating outbound channel: "+err.Error())
		return err
	}

	// create kafka producer
	producer, err := CreateKafkaProducer()
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return err
	}

	helpers.Log(logrus.DebugLevel, "Originating call...")

	params := types.CallParams{
		From:        callerId,
		To:          numberToCall,
		Status:      "start",
		Direction:   "outbound",
		UserId:      user.Id,
		WorkspaceId: user.Workspace.Id,
		ChannelId:   outboundChannel.ID()}
	body, err := json.Marshal(params)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return err
	}

	helpers.Log(logrus.InfoLevel, "creating call...")
	helpers.Log(logrus.InfoLevel, "calling "+numberToCall)
	resp, err := api.SendHttpRequest("/call/createCall", body)

	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return err
	}
	id := resp.Headers.Get("x-call-id")
	helpers.Log(logrus.DebugLevel, "Call ID is: "+id)
	idAsInt, err := strconv.Atoi(id)

	call := types.Call{
		CallId:  idAsInt,
		Channel: lineChannel,
		Started: time.Now(),
		Params:  &params}

	domain := user.Workspace.Domain
	apiCallId := strconv.Itoa(call.CallId)
	headers := CreateSIPHeaders(domain, callerId, typeOfCall, apiCallId, addedHeaders)
	outboundChannel, err = outboundChannel.Originate(CreateOriginateRequest(callerId, numberToCall, headers))
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return err
	}

	stopChannel := make(chan bool)
	outChannel.Channel = outboundChannel
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go manageBridge(&lineBridge, &call, user, lineChannel, &outChannel, producer, wg)
	wg.Wait()
	if err := bridge.AddChannel(lineChannel.Channel.Key().ID); err != nil {
		helpers.Log(logrus.ErrorLevel, "failed to add channel to bridge"+" error:"+err.Error())
		return errors.New("failed to add channel to bridge")
	}

	helpers.Log(logrus.InfoLevel, "creating outbound call...")
	//resp, err = api.SendHttpRequest("/call/createCall", body)
	//_, err = outChannel.CreateCall(resp.Headers.Get("x-call-id"), &params)
	_, err = outChannel.CreateCall(id, &params)

	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return err
	}

	lineChannel.Channel.Ring()
	wg1 := new(sync.WaitGroup)
	wg1.Add(1)
	lineBridge.AddChannel(lineChannel)
	lineBridge.AddChannel(&outChannel)
	go manageOutboundCallLeg(lineChannel, &outChannel, &lineBridge, &call, wg1, stopChannel)
	wg1.Wait()

	//
	timeout := 90
	wg2 := new(sync.WaitGroup)
	wg2.Add(1)
	go lineBridge.StartRingTimer(timeout, wg2, stopChannel)
	wg2.Wait()

	return nil
}

func manageBridge(bridge *types.LineBridge, call *types.Call, user *types.User, lineChannel *types.LineChannel, outboundChannel *types.LineChannel, producer *kafka.Producer, wg *sync.WaitGroup) {
	h := bridge.Bridge

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

	storageServer := types.StorageServer{
		Ip: GetARIHost(),
	}

	record := processor_helpers.NewRecording(&storageServer, producer, user, &call.CallId, false)
	//_,recordErr:=record.InitiateRecordingForBridge(bridge)
	_, recordErr := record.InitiateRecordingForBridge(bridge)
	if recordErr != nil {
		helpers.Log(logrus.ErrorLevel, "error starting recording: "+recordErr.Error())
		return
	}

	var numChannelsEntered int = 0
	for {
		select {
		case <-destroySub.Events():
			helpers.Log(logrus.DebugLevel, "bridge destroyed")
			return
		case e, ok := <-enterSub.Events():
			if !ok {
				helpers.Log(logrus.ErrorLevel, "channel entered subscription closed")
				return
			}

			v := e.(*ari.ChannelEnteredBridge)
			numChannelsEntered += 1

			helpers.Log(logrus.DebugLevel, "channel entered bridge "+"channel "+v.Channel.Name)
			helpers.Log(logrus.DebugLevel, "num channels in bridge: "+strconv.Itoa(numChannelsEntered))

		case e, ok := <-leaveSub.Events():
			if !ok {
				helpers.Log(logrus.ErrorLevel, "channel left subscription closed")
				return
			}
			v := e.(*ari.ChannelLeftBridge)
			helpers.Log(logrus.DebugLevel, "channel left bridge"+" channel "+v.Channel.Name)
			helpers.Log(logrus.DebugLevel, "ending all calls in bridge...")
			// end both calls
			lineChannel.SafeHangup()
			outboundChannel.SafeHangup()
			record.Stop()

			helpers.Log(logrus.DebugLevel, "updating call status...")
			api.UpdateCall(call, "ended")
		}
	}
}

func manageOutboundCallLeg(lineChannel *types.LineChannel, outboundChannel *types.LineChannel, lineBridge *types.LineBridge, call *types.Call, wg *sync.WaitGroup, ringTimeoutChan chan<- bool) error {

	endSub := outboundChannel.Channel.Subscribe(ari.Events.StasisEnd)
	defer endSub.Cancel()
	startSub := outboundChannel.Channel.Subscribe(ari.Events.StasisStart)

	defer startSub.Cancel()
	destroyedSub := outboundChannel.Channel.Subscribe(ari.Events.ChannelDestroyed)
	defer destroyedSub.Cancel()
	wg.Done()
	helpers.Log(logrus.DebugLevel, "managing outbound call...")
	helpers.Log(logrus.DebugLevel, "listening for channel events...")

	for {

		select {
		case <-startSub.Events():
			helpers.Log(logrus.DebugLevel, "started call..")

			if err := lineBridge.Bridge.AddChannel(outboundChannel.Channel.Key().ID); err != nil {
				helpers.Log(logrus.ErrorLevel, "failed to add channel to bridge"+" error:"+err.Error())
				return err
			}
			helpers.Log(logrus.DebugLevel, "added outbound channel to bridge..")
			lineChannel.Channel.StopRing()
			ringTimeoutChan <- true
		case <-endSub.Events():
			helpers.Log(logrus.DebugLevel, "ended call..")
			lineChannel.Channel.StopRing()
			lineChannel.Channel.Hangup()
			//lineBridge.EndBridgeCall()
			helpers.Log(logrus.DebugLevel, "manageOutboundCallLeg ended call..")
			helpers.Log(logrus.DebugLevel, "manageOutboundCallLeg stasis end called..")
			call.Ended = time.Now()
			params := types.StatusParams{
				CallId: call.CallId,
				Ip:     GetPublicIp(),
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
		case <-destroyedSub.Events():
			helpers.Log(logrus.DebugLevel, "channel destroyed..")
			lineChannel.Channel.StopRing()
			lineChannel.Channel.Hangup()
			//lineBridge.EndBridgeCall()

		}
	}
}

func ProcessSIPTrunkCall(
	cl ari.Client,
	src *ari.Key,
	user *types.User,
	lineChannel *types.LineChannel,
	callerId string,
	exten string,
	trunkAddr string) error {
	helpers.Log(logrus.DebugLevel, "ensureBridge called..")
	var bridge *ari.BridgeHandle
	var err error
	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
	bridge, err = cl.Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		bridge = nil
		return eris.Wrap(err, "failed to create bridge")
	}
	outChannel := types.LineChannel{}
	lineBridge := types.NewBridge(bridge, false)

	helpers.Log(logrus.InfoLevel, "channel added to bridge")
	outboundChannel, err := cl.Channel().Create(nil, CreateChannelRequest(exten))

	if err != nil {
		helpers.Log(logrus.DebugLevel, "error creating outbound channel: "+err.Error())
		return err
	}

	helpers.Log(logrus.DebugLevel, "Originating call...")

	params := types.CallParams{
		From:        callerId,
		To:          exten,
		Status:      "start",
		Direction:   "inbound",
		UserId:      user.Id,
		WorkspaceId: user.Workspace.Id,
		ChannelId:   outboundChannel.ID()}
	body, err := json.Marshal(params)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return err
	}

	helpers.Log(logrus.InfoLevel, "creating call...")
	helpers.Log(logrus.InfoLevel, "calling "+exten)
	resp, err := api.SendHttpRequest("/call/createCall", body)

	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return err
	}
	id := resp.Headers.Get("x-call-id")
	helpers.Log(logrus.DebugLevel, "Call ID is: "+id)
	idAsInt, err := strconv.Atoi(id)

	call := types.Call{
		CallId:  idAsInt,
		Channel: lineChannel,
		Started: time.Now(),
		Params:  &params}

	domain := user.Workspace.Domain
	apiCallId := strconv.Itoa(call.CallId)
	headers := CreateSIPHeadersForSIPTrunkCall(domain, callerId, "pstn", apiCallId, trunkAddr)
	outboundChannel, err = outboundChannel.Originate(CreateOriginateRequest(callerId, exten, headers))
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return err
	}

	// create kafka producer
	producer, err := CreateKafkaProducer()
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return err
	}

	stopChannel := make(chan bool)
	outChannel.Channel = outboundChannel
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go manageBridge(&lineBridge, &call, user, lineChannel, &outChannel, producer, wg)
	wg.Wait()
	if err := bridge.AddChannel(lineChannel.Channel.Key().ID); err != nil {
		helpers.Log(logrus.ErrorLevel, "failed to add channel to bridge"+" error:"+err.Error())
		return errors.New("failed to add channel to bridge")
	}

	helpers.Log(logrus.InfoLevel, "creating outbound call...")
	resp, err = api.SendHttpRequest("/call/createCall", body)
	_, err = outChannel.CreateCall(resp.Headers.Get("x-call-id"), &params)

	if err != nil {
		helpers.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		return err
	}

	lineChannel.Channel.Ring()
	wg1 := new(sync.WaitGroup)
	wg1.Add(1)
	lineBridge.AddChannel(lineChannel)
	lineBridge.AddChannel(&outChannel)
	go manageOutboundCallLeg(lineChannel, &outChannel, &lineBridge, &call, wg1, stopChannel)
	wg1.Wait()

	timeout := 30
	wg2 := new(sync.WaitGroup)
	wg2.Add(1)
	go lineBridge.StartRingTimer(timeout, wg2, stopChannel)
	wg2.Wait()

	return nil
}

func DeviceStateChangeHandler(m map[string]string) {
	fmt.Printf("DeviceStateChange event received: %v\n", m)
}

func DefaultHandler(m map[string]string) {
	fmt.Printf("Event received: %v\n", m)
}

func CreateAMIClient() (*amigo.Amigo, error) {
	helpers.Log(logrus.InfoLevel, fmt.Sprintf("logging into AMI host at %s", os.Getenv("AMI_HOST")))
	settings := &amigo.Settings{
		Username: os.Getenv("AMI_USER"),
		Password: os.Getenv("AMI_PASS"),
		Host: os.Getenv("AMI_HOST"),
	}
	a := amigo.New(settings)

	a.Connect()

	// Listen for connection events
	a.On("connect", func(message string) {
		fmt.Println("connected to AMI. " + message)
	})
	a.On("error", func(message string) {
		fmt.Println("AMI error occured: " + message)
	})

	// Registering handler function for event "DeviceStateChange"
	a.RegisterHandler("DeviceStateChange", DeviceStateChangeHandler)
	// Registering default handler function for all events.
	a.RegisterDefaultHandler(DefaultHandler)

	// Optionally create channel to receiving all events
	// and set created channel to receive all events
	//c := make(chan map[string]string, 100)
	//a.SetEventChannel(c)

	// Check if connected with Asterisk, will send Action "QueueSummary"
	// TODO: report errors when we can't connect to AMI
	// if a.Connected() {
	// 	return a,nil
	// }

	return a,nil
}

/*
Config func to get env value from key ---
*/
func Config(key string) string {
	// load .env file
	loadDotEnv := os.Getenv("USE_DOTENV")
	if loadDotEnv != "off" {
		err := godotenv.Load(".env")
		if err != nil {
			fmt.Print("Error loading .env file")
		}
	}
	return os.Getenv(key)
}

func CreateCallDebit(callChannel *types.Call) error {
	return nil
}
