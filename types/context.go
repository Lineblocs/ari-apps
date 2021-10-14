package types
import (
	"github.com/inconshreveable/log15"
	"github.com/CyCoreSystems/ari/v5"
	"context"
	"strings"
	"regexp"
	"fmt"
) 

type Context struct {
	Flow *Flow
	Cell *Cell
	Channel *LineChannel
	Runner *Runner
	Vars *FlowVars
 	Log log15.Logger
 	Context context.Context
	 Client ari.Client
	RecvChannel chan<- Link
}

func convertVariableValues(value string, lineFlow *Flow) (string) {
    rex := regexp.MustCompile("\\{\\{[\\w\\d\\.]+\\}\\}")
	out := rex.FindAllStringSubmatch(value, -1)

    for _, i := range out {
		match := i[1]
        fmt.Println(match)
		rex1 := regexp.MustCompile("^\\{\\{|\\}\\}$")
		updated := strings.Split(rex1.ReplaceAllString(match, ""), ".")
		if len( updated ) > 2 {
			rex2 := regexp.MustCompile(match)
    		value = rex2.ReplaceAllString(value, "")
		} else {
			rex2 := regexp.MustCompile(match)
    		value = rex2.ReplaceAllString(value, "")
		}
    }
	return value
}
func processInterpolation( value* ModelData, lineFlow *Flow) {

	if value.IsStr {
		value.ValueStr = convertVariableValues(value.ValueStr, lineFlow)
	} else if value.IsObj {
		for k, v := range value.ValueObj {
			value.ValueObj[ k ] = convertVariableValues(v, lineFlow)
		}

	} else if value.IsArray {
		for k, v := range value.ValueArr {
			value.ValueArr[ k ] = convertVariableValues(v, lineFlow)
		}
	}
}
func processAllInterpolations( data map[string] ModelData, lineFlow *Flow) {
	for key, val := range data {
		interoplated := "_before_interopolations"
		if strings.HasSuffix(key, interoplated) {
			continue
		}
		interoplatedKey := key+"_before_interopolations"
		if before, ok := data[interoplatedKey]; ok {
		    //do something here
			processInterpolation(&before, lineFlow)
			continue
		}
		processInterpolation(&val, lineFlow)
	}
}
func NewContext(cl ari.Client, ctx context.Context, recvChannel chan<- Link,log *log15.Logger, flow *Flow, cell *Cell, runner *Runner, channel *LineChannel) (*Context) {
	processAllInterpolations( cell.Model.Data, flow );
	return &Context{Client: cl, Log: *log, Context: ctx, Channel: channel, Cell: cell, Flow: flow, Runner: runner, RecvChannel: recvChannel};
}