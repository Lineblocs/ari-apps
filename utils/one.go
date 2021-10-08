package utils

import (
	"errors"
	"strconv"
	"time"
	"lineblocs.com/processor/types"
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

func GetSIPProxy() (string) {
	//return "proxy1";
	return "127.0.0.1"
}