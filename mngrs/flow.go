package mngrs

import (
	"github.com/CyCoreSystems/ari/v5"
	"context"
	"strconv"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)

func startProcessingFlow( cl ari.Client, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, eventVars map[string] string, cell *types.Cell, runner *types.Runner) {
	log := utils.GetLogger()
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
			mngr := NewBridgeManager(lineCtx, flow)
			mngr.StartProcessing()
		case "devs.DialModel":
		default:
	}
}


func ProcessFlow( cl ari.Client, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, eventVars map[string] string, cell *types.Cell) {
	log := utils.GetLogger()
	log.Debug("processing cell type " + cell.Cell.Type)
	runner:=types.Runner{Cancelled: false}
	flow.Runners = append( flow.Runners, &runner )
	startProcessingFlow( cl, ctx, flow, lineChannel, eventVars, cell, &runner)
}