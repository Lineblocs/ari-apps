package utils

import (
	"errors"
	"strconv"
	"time"
	"lineblocs.com/processor/types"
	"github.com/CyCoreSystems/ari/v5"
)
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
		return data["phone_number"].ValueStr
	}
	return ""
}

func SafeHangup(lineChannel *types.LineChannel) {
	if lineChannel.Channel != nil {
		lineChannel.Channel.Hangup()
	}
}



func GetSIPSecretKey() string {
	return "xxx-1"
}


func CreateSIPHeaders(domain string, callerId string, typeOfCall string) map[string]string {
	headers := make( map[string]string )
	headers["SIPADDHEADER0"] = "X-LineBlocs-Key: " + GetSIPSecretKey()
	headers["SIPADDHEADER1"] = "X-LineBlocs-Domain: " + domain
	headers["SIPADDHEADER3"] = "X-LineBlocs-Route-Type: " + typeOfCall
	headers["SIPADDHEADER4"] ="X-LineBlocs-Caller: " + callerId
	return headers
}