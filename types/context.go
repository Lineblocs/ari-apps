package types
import (
	"github.com/inconshreveable/log15"
	"context"
	"strings"
)

type Context struct {
	Flow *Flow
	Cell *Cell
	Channel *LineChannel
	Runner *Runner
	Vars *FlowVars
 	Log log15.Logger
 	Context context.Context
}

func processInterpolation( value ModelData, lineFlow *Flow) (ModelData) {
	return ModelData{}
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
			data[ key ] = processInterpolation(before, lineFlow)
			continue
		}
		data[ key ] = processInterpolation(val, lineFlow)
	}
}
func NewContext(ctx context.Context, log *log15.Logger, flow *Flow, cell *Cell, runner *Runner, channel *LineChannel) (*Context) {
	processAllInterpolations( cell.Model.Data, flow );
	return &Context{Log: *log, Context: ctx, Channel: channel, Cell: cell, Flow: flow, Runner: runner};
}