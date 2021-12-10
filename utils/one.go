package utils

import (
	"errors"
	"strconv"
	"time"
	"context"
	"strings"
	"io/ioutil"
	"net/http"
	"encoding/json"
	"os"
	"io"
	"fmt"
	"path"
	"lineblocs.com/processor/types"
	"golang.org/x/oauth2/google"
	"github.com/CyCoreSystems/ari/v5"
	"github.com/inconshreveable/log15"
	"github.com/google/uuid"
	"github.com/u2takey/ffmpeg-go"
	    "github.com/aws/aws-sdk-go/aws"
    "github.com/aws/aws-sdk-go/aws/session"
    "github.com/aws/aws-sdk-go/aws/credentials"
	    "github.com/aws/aws-sdk-go/service/s3/s3manager"
		    "github.com/go-redis/redis/v8"
			"google.golang.org/api/option"
	        texttospeech "cloud.google.com/go/texttospeech/apiv1"
        texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
        speech "cloud.google.com/go/speech/apiv1"
        speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
	"github.com/CyCoreSystems/ari/v5/ext/record"
	"lineblocs.com/processor/api"
)


var log log15.Logger
type ConfCache struct {
	Id string `json:"id"`	
	BridgeId string `json:"bridgeId"`	
	UserInfo *types.UserInfo `json:"userInfo"`	
}
// TODO get the ip
func GetPublicIp( ) string {
	return "0.0.0.0"
}

func PlaybackLoops( data types.ModelData  ) (int) {
	item,ok := data.(types.ModelDataStr)

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
func DetermineCallerId( call *types.Call, data types.ModelData  ) (string) {
	item,ok := data.(types.ModelDataStr)

	if !ok {
		return call.Params.From
	}
	if item.Value == "" {
		// default caller id
		return call.Params.From
	}
	return item.Value
}

func CheckFreeTrial( plan string ) bool {
	if plan == "expired" {
		return true
	}
	return false
}

func CreateRDB() (*redis.Client) {
    rdb := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "", // no password set
        DB:       0,  // use default DB
    })
	return rdb
}

func FindLinkByName( links []*types.Link, direction string, tag string) (*types.Link, error) {
		fmt.Println("FindLinkByName called...")
	for _, link := range links {
		fmt.Println("FindLinkByName checking source port: " + link.Link.Source.Port)
		fmt.Println("FindLinkByName checking target port: " + link.Link.Target.Port)
		if direction == "source" {
			fmt.Println("FindLinkByName checking link: " + link.Source.Cell.Name)
			if link.Link.Source.Port == tag {
				return link, nil
			}
		} else if direction == "target" {

			fmt.Println("FindLinkByName checking link: " + link.Target.Cell.Name)
			if link.Link.Target.Port == tag {
				return link, nil
			}
		}
	}
	return nil, errors.New("Could not find link")
}
func GetCellByName( flow *types.Flow, name string ) (*types.Cell, error) {
	for _, v := range flow.Cells {

		if v.Cell.Name == name {
			return v, nil
		}
	}
	return nil, nil
}
func LookupCellVariable( flow *types.Flow, name string, lookup string) (string, error) {
	var cell *types.Cell
	cell, err := GetCellByName( flow, name )
	if err != nil {
		return "", err
	}
	if cell == nil {
		return "", errors.New("Could not find cell")
	}
	fmt.Println("looking up cell variable\r\n");
	fmt.Println(cell.Cell.Type);
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
			return strconv.Itoa( call.GetStartTime() ), nil
		} else if lookup == "ended" {
			call := cell.AttachedCall
			return strconv.Itoa( call.FigureOutEndedTime() ), nil
		}
	} else if cell.Cell.Type == "devs.ProcessInputModel" {
		fmt.Println("getting input value..\r\n");
		if lookup == "digits" {
			fmt.Println("found:");
			fmt.Println( cell.EventVars["digits"] );
			return cell.EventVars["digits"], nil
		}
	}
	return "", errors.New("Could not find link")
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
	//return "52.60.126.237"
	//return "159.89.124.168"
	return os.Getenv("PROXY_HOST")
}

func GetARIHost() (string) {
	return os.Getenv("ARI_HOST")
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

func DetermineNumberToCall(data map[string]types.ModelData) (string, error) {
	callType, ok := data["call_type"].(types.ModelDataStr)
	if !ok {
		return "", errors.New("Could not get call type")
	}

	switch ; callType.Value {
	case "Extension":
		ext, ok := data["extension"].(types.ModelDataStr)
		if !ok {
			return "", errors.New("Could not get ext")
		}
		return ext.Value,nil
	case "Phone Number":
		ext, ok := data["number_to_call"].(types.ModelDataStr)
		if !ok {
			return "", errors.New("Could not get number")
		}
		return ext.Value,nil
	}
	return "", errors.New("Unknown call type")
}

func SafeHangup(lineChannel *types.LineChannel) {
	if lineChannel.Channel != nil {
		lineChannel.Channel.Hangup()
	}
}



func GetSIPSecretKey() string {
	//return "BrVIsXzQx9-7lvRsXMC2V57dA4UEc-G_HwnCpK-zctk"
	//return "BrVIsXzQx9-7lvRsXMC2V57dA4UEc-G_HwnCpK-zctk"
	return "xxx"
}


func CreateSIPHeaders(domain, callerId, typeOfCall, apiCallId string) map[string]string {
	headers := make( map[string]string )
	headers["SIPADDHEADER0"] = "X-LineBlocs-Key: " + GetSIPSecretKey()
	headers["SIPADDHEADER1"] = "X-LineBlocs-Domain: " + domain
	headers["SIPADDHEADER2"] = "X-LineBlocs-Route-Type: " + typeOfCall
	headers["SIPADDHEADER3"] ="X-LineBlocs-Caller: " + callerId
	headers["SIPADDHEADER4"] ="X-LineBlocs-API-CallId: " + apiCallId
	return headers
}

func GetLogger() (log15.Logger) {
	if log == nil {
 		newLog := log15.New()
		 log =newLog
	}
	return log
}

func sendToAssetServer( path string, filename string ) (string, error) {
	settings, err := api.GetSettings()
	if err != nil {
		return "", err
	}

	creds := credentials.NewStaticCredentials(
		settings.AwsAccessKeyId,
		settings.AwsSecretAccessKey, "")

	sess, err := session.NewSession(&aws.Config{
    		Region:      aws.String(settings.AwsRegion),
    		Credentials: creds,
	})
	if err != nil {
		return "", fmt.Errorf("error occured: %v", err)
	}


	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)

	f, err  := os.Open(path)
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
	fmt.Printf("file uploaded to, %s\n", aws.StringValue(&result.Location))


	// send back link to media
	url := "https://lineblocs.s3.ca-central-1.amazonaws.com/" + key
	return url, nil
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
		log.Error(err.Error())
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

	fullPathToFile, err := changeAudioEncoding( filepath, ext )
	if err != nil {
		return "", err
	}

	link, err  := sendToAssetServer( fullPathToFile, filename )
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
	var serviceAccountKey = []byte(settings.GoogleServiceAccountJson)
 
	creds, err := google.CredentialsFromJSON(ctx, serviceAccountKey)
	if err != nil {
		log.Error(err.Error())
		return "", err
	}
	ctx2 := context.Background()
	//client, err := texttospeech.NewClient(ctx)
	opt := option.WithCredentials(creds)
	client, err := texttospeech.NewClient(ctx2, opt)
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
	var folder string = "/tmp/"
	uniq, err := uuid.NewUUID()
	if err != nil {
		log.Error(err.Error())
		return "", err
	}

 	filename := (uniq.String() + ".wav")
	fullPathToFile := folder + filename

	err = ioutil.WriteFile(fullPathToFile, resp.AudioContent, 0644)
	if err != nil {
			log.Error(err.Error())
			return "", err
	}
	fmt.Printf("Audio content written to file: %v\n", fullPathToFile)
	link, err  := sendToAssetServer(  fullPathToFile, filename )
	if err != nil {
		return "", err
	}


	return link, nil
}

func StartSTT(fileURI string) (string, error) {
		ctx:= context.Background()
		settings, err := api.GetSettings()
		if err != nil {
			return "", err
		}
		var serviceAccountKey = []byte(settings.GoogleServiceAccountJson)
	
		creds, err := google.CredentialsFromJSON(ctx, serviceAccountKey)
		if err != nil {
			log.Error(err.Error())
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
		var highestConfidence float32 =0.0
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
		log.Error(err.Error())
		return "", err
	}

	data := []byte("")
 	filename := (uniq.String() + ".wav")
	fullPathToFile := folder + filename

	err = ioutil.WriteFile(fullPathToFile, data, 0644)
	if err != nil {
			log.Error(err.Error())
			return "", err
	}
	fmt.Printf("Audio content written to file: %v\n", fullPathToFile)
	link, err := sendToAssetServer(  fullPathToFile, filename )
	if err != nil {
		return "", err
	}
	return link, nil
}



func changeAudioEncoding(filepath string, ext string) (string, error) {
	newfile := filepath + ".wav"


	err := ffmpeg_go.Input(filepath).Output(newfile, ffmpeg_go.KwArgs{
			"acodec": "pcm_u8",
			"ar": "8000",
	}).OverWriteOutput().Run()

	if err != nil {
		return "", err
	}
	return newfile, nil

}

func AddChannelToBridge( bridge *types.LineBridge, channel *types.LineChannel) {
	bridge.Channels = append( bridge.Channels, channel )
}

func RemoveChannelFromBridge( bridge *types.LineBridge, channel *types.LineChannel) {
	channels := make([]*types.LineChannel, 0)
	for _, item := range bridge.Channels {
		if item.Channel.ID() != channel.Channel.ID() {
			channels = append( channels, item )
		}
	}
	bridge.Channels =channels
}

func ParseRingTimeout( value types.ModelData ) (int) {

	item,ok := value.(types.ModelDataStr)
	if !ok {
		return 30
	}
	result, err := strconv.Atoi( item.Value )

	// use default
	if err != nil {
		return 30
	}
	return result

}

func SafeSendResonseToChannel(channel chan<- *types.ManagerResponse, resp *types.ManagerResponse) {
}

func GetWorkspaceNameFromDomain(domain string) (string) {
	s := strings.Split(domain, ".")
	return s[0]
}


func AddConfBridge( client ari.Client, workspace string, confName string, conf *types.LineConference ) (*types.LineConference, error) {
	var ctx = context.Background()
 	key := workspace + "_" + confName
	rdb := CreateRDB()
	params := ConfCache{
		Id: conf.Id,
		UserInfo: &conf.User.Info,
		BridgeId:  conf.Bridge.Bridge.ID() }
	body, err := json.Marshal( params )
	if err != nil {
		log.Error( "error occured: " + err.Error() )
		return nil, err
	}

    err = rdb.Set(ctx, key, body, 0).Err()
    if err != nil {
		return nil, err
    }

	return conf, nil
}


func GetConfBridge( client ari.Client, user *types.User, confName string ) (*types.LineConference, error) {
	var ctx = context.Background()
 	key := strconv.Itoa( user.Workspace.Id ) + "_" + confName
	rdb := CreateRDB()
    val, err := rdb.Get(ctx, key).Result()
    if err != nil {
		return nil, err
    }
    fmt.Println("key", val)
	var data ConfCache
	err = json.Unmarshal( []byte(val), &data )
    if err != nil {
		return nil, err
	}
	src := ari.NewKey(ari.BridgeKey, data.BridgeId)
	bridge := client.Bridge().Get(src)
	conf := types.NewConference(data.Id, user, &types.LineBridge{ Bridge: bridge })
	return conf, nil
}