package utils

import (
	"errors"
	"strconv"
	"time"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"io"
	"fmt"
	"lineblocs.com/processor/types"
	"github.com/CyCoreSystems/ari/v5"
	"github.com/inconshreveable/log15"
	"github.com/google/uuid"
		_ "github.com/krig/go-sox"
	        texttospeech "cloud.google.com/go/texttospeech/apiv1"
        texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
)


var log log15.Logger

// TODO get the ip
func GetPublicIp( ) string {
	return "0.0.0.0"
}
func DetermineCallerId( call *types.Call, providedVal string ) (string) {
	if providedVal == "" {
		// default caller id
		return call.Params.From
	}
	return providedVal
}

func CheckFreeTrial( plan string ) bool {
	if plan == "expired" {
		return true
	}
	return false
}

func FindLinkByName( links []*types.Link, direction string, tag string) (*types.Link, error) {
	for _, link := range links {
		if direction == "source" {
			if link.Source.Cell.Source.Port == tag {
				return link, nil
			}
		} else if direction == "target" {
			if link.Target.Cell.Target.Port == tag {
				return link, nil
			}
		}
	}
	return nil, errors.New("Could not find link")
}

func CreateCall( id string, channel *types.LineChannel, params *types.CallParams) (*types.Call, error) {
		idAsInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, err 
	}

	call := types.Call{
		CallId: idAsInt,
		Channel: channel,
		Started: time.Now(),
		Params: params }
	return &call, nil
}

// TODO call API to get proxy IPs
func GetSIPProxy() (string) {
	//return "proxy1";
	return "52.60.126.237"
}

func CreateChannelRequest(numberToCall string) (ari.ChannelCreateRequest) {
 	return ari.ChannelCreateRequest{
		Endpoint: "SIP/" + numberToCall + "@" + GetSIPProxy(),
		App:      "lineblocs",
		AppArgs: "DID_DIAL," }
}

func CreateChannelRequest2(numberToCall string) (ari.ChannelCreateRequest) {
 	return ari.ChannelCreateRequest{
		Endpoint: "SIP/" + numberToCall + "/" + GetSIPProxy(),
		App:      "lineblocs",
		AppArgs: "DID_DIAL_2," }
}



func CreateOriginateRequest(callerId string, numberToCall string, headers map[string] string) (ari.OriginateRequest) {
 	return ari.OriginateRequest{
		CallerID: callerId,
		Endpoint: "SIP/" + numberToCall + "@" + GetSIPProxy(),
		App: "lineblocs",
		AppArgs: "DID_DIAL,", Variables: headers }
}

func CreateOriginateRequest2(callerId string, numberToCall string) (ari.OriginateRequest) {
 	return ari.OriginateRequest{
		CallerID: callerId,
		Endpoint: "SIP/" + numberToCall + "/" + GetSIPProxy(),
		App: "lineblocs",
		AppArgs: "DID_DIAL_2," }
}

func DetermineNumberToCall(data map[string]types.ModelData) (string) {
	callType := data["call_type"]

	if callType.ValueStr == "Extension" {
		return data["extension"].ValueStr
	} else if callType.ValueStr == "Phone Number" {
		return data["number_to_call"].ValueStr
	}
	return ""
}

func SafeHangup(lineChannel *types.LineChannel) {
	if lineChannel.Channel != nil {
		lineChannel.Channel.Hangup()
	}
}



func GetSIPSecretKey() string {
	//return "BrVIsXzQx9-7lvRsXMC2V57dA4UEc-G_HwnCpK-zctk"
	return "BrVIsXzQx9-7lvRsXMC2V57dA4UEc-G_HwnCpK-zctk"
}


func CreateSIPHeaders(domain string, callerId string, typeOfCall string) map[string]string {
	headers := make( map[string]string )
	headers["SIPADDHEADER0"] = "X-LineBlocs-Key: " + GetSIPSecretKey()
	headers["SIPADDHEADER1"] = "X-LineBlocs-Domain: " + domain
	headers["SIPADDHEADER2"] = "X-LineBlocs-Route-Type: " + typeOfCall
	headers["SIPADDHEADER3"] ="X-LineBlocs-Caller: " + callerId
	return headers
}

func GetLogger() (log15.Logger) {
	if log == nil {
 		newLog := log15.New()
		 log =newLog
	}
	return log
}


func DownloadFile(flow *types.Flow, url string) (string, error) {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var folder string = "/var/lib/asterisk/sounds/en/lineblocs/"
	uniq, err := uuid.NewUUID()
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	var filename = url
	var ext = filepath.Ext(filename)
	//var name = filename[0:len(filename)-len(extension)]

	filepath := folder + (uniq.String() + "." + ext)
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

	path, err := changeAudioEncoding( filepath, ext )
	if err != nil {
		return "", err
	}
	

	return "", err
}

func StartTTS(flow *types.Flow, say string, gender string, voice string, lang string) (string, error) {
	// Instantiates a client.
	ctx := context.Background()

	client, err := texttospeech.NewClient(ctx)
	if err != nil {
			log.Error(err.Error())
			return "", err
	}
	defer client.Close()

	var ssmlGender texttospeechpb.SsmlVoiceGender
	if gender == "MALE" {
		ssmlGender =  texttospeechpb.SsmlVoiceGender_MALE
	} else if gender == "FEMALE" {
		ssmlGender =  texttospeechpb.SsmlVoiceGender_FEMALE
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
				Name: voice,
					LanguageCode: lang,
					//SsmlGender:   texttospeechpb.SsmlVoiceGender_NEUTRAL,
					SsmlGender:   ssmlGender,
			},
			// Select the type of audio file you want returned.
			AudioConfig: &texttospeechpb.AudioConfig{
					//AudioEncoding: texttospeechpb.AudioEncoding_MP3,
					AudioEncoding: texttospeechpb.AudioEncoding_LINEAR16,
					SampleRateHertz: 8000,
			},
	}

	resp, err := client.SynthesizeSpeech(ctx, &req)
	if err != nil {
			log.Error(err.Error())
			return "", err
	}

	// The resp's AudioContent is binary.
	var folder string = "/var/lib/asterisk/sounds/en/lineblocs/"
	uniq, err := uuid.NewUUID()
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

	filename := folder + (uniq.String() + ".wav")

	err = ioutil.WriteFile(filename, resp.AudioContent, 0644)
	if err != nil {
			log.Error(err.Error())
			return "", err
	}
	fmt.Printf("Audio content written to file: %v\n", filename)
	return "", nil
}



func changeAudioEncoding(filepath string, ext string) (string, error) {
	channel := 1
	newfile = filepath + ".wav"


	err := ffmpeg_go.Input(filepath).Output(newfile, ffmpeg_go.KwArgs{
			"acodec": "pcm_u8",
			"ar": "8000",
	}).OverWriteOutput().Run()

	if err != nil {
		return "", err
	}
	return newfile, nil

}