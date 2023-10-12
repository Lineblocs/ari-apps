package types

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/ivahaev/amigo"
)

type Context struct {
	Flow        *Flow
	Cell        *Cell
	Channel     *LineChannel
	Runner      *Runner
	Vars        *FlowVars
	Context     context.Context
	Client      ari.Client
	AMIClient   *amigo.Amigo
	RecvChannel chan<- *ManagerResponse
}

func convertVariableValues(value string, lineFlow *Flow) string {
	rex := regexp.MustCompile("\\{\\{[\\w\\d\\.]+\\}\\}")
	out := rex.FindAllStringSubmatch(value, -1)

	for _, i := range out {
		fmt.Println(i)
		match := i[0]
		fmt.Println(match)
		rex1 := regexp.MustCompile("^\\{\\{|\\}\\}$")
		updated := strings.Split(rex1.ReplaceAllString(match, ""), ".")
		if len(updated) > 2 {
			rex2 := regexp.MustCompile(match)
			value = rex2.ReplaceAllString(value, "")
		} else {
			rex2 := regexp.MustCompile(match)
			value = rex2.ReplaceAllString(value, "")
		}
	}
	return value
}
func processInterpolation(i ModelData, lineFlow *Flow) {
	itemStr, ok := i.(ModelDataStr)
	if ok {
		itemStr.Value = convertVariableValues(itemStr.Value, lineFlow)
		return
	}
	itemObj, ok := i.(ModelDataObj)
	if ok {
		for k, v := range itemObj.Value {
			itemObj.Value[k] = convertVariableValues(v, lineFlow)
		}
		return
	}
	itemArr, ok := i.(ModelDataArr)
	if ok {
		for k, v := range itemArr.Value {
			itemArr.Value[k] = convertVariableValues(v, lineFlow)
		}
		return
	}
}
func processAllInterpolations(data map[string]ModelData, lineFlow *Flow) {
	for key, val := range data {
		interoplated := "_before_interopolations"
		if strings.HasSuffix(key, interoplated) {
			continue
		}
		interoplatedKey := key + "_before_interopolations"
		if before, ok := data[interoplatedKey]; ok {
			//do something here
			processInterpolation(&before, lineFlow)
			continue
		}
		processInterpolation(&val, lineFlow)
	}
}
func NewContext(cl ari.Client, amiClient *amigo.Amigo, ctx context.Context, recvChannel chan<- *ManagerResponse, flow *Flow, cell *Cell, runner *Runner, channel *LineChannel) *Context {
	processAllInterpolations(cell.Model.Data, flow)
	return &Context{
		Client: cl, 
		AMIClient: amiClient,
		Context: ctx, 
		Channel: channel, 
		Cell: cell, 
		Flow: flow, 
		Runner: runner, 
		RecvChannel: recvChannel,
	}
}
