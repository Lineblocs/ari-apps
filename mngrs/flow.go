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

	manRecvChannel := make(chan *types.ManagerResponse)	
	lineCtx := types.NewContext(
		cl,
		ctx,
		manRecvChannel,
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
			mngr := NewSwitchManager(lineCtx, flow)
			mngr.StartProcessing()
		case "devs.BridgeModel":
			mngr := NewBridgeManager(lineCtx, flow)
			mngr.StartProcessing()
		case "devs.PlaybackModel":
			mngr := NewPlaybackManager(lineCtx, flow)
			mngr.StartProcessing()
		case "devs.ProcessInputModel":
			mngr := NewInputManager(lineCtx, flow)
			mngr.StartProcessing()
		case "devs.DialModel":
			mngr := NewDialManager(lineCtx, flow)
			mngr.StartProcessing()
		case "devs.SetVariablesModel":
			mngr := NewSetVariablesManager(lineCtx, flow)
			mngr.StartProcessing()
		case "devs.WaitModel":
			mngr := NewWaitManager(lineCtx, flow)
			mngr.StartProcessing()
		case "devs.SendDigitsModel":
			mngr := NewSendDigitsManager(lineCtx, flow)
			mngr.StartProcessing()
		case "devs.MacroModel":
			mngr := NewMacroManager(lineCtx, flow)
			mngr.StartProcessing()
		case "devs.ConferenceModel":
			mngr := NewConferenceManager(lineCtx, flow)
			mngr.StartProcessing()
		default:

	}

	log.Debug("waiting to receive from channel...")
	for {
		select {
			case resp, ok := <-manRecvChannel:
				if !ok {
					log.Debug("error receiving result from cell..")
					return
				}
				log.Debug("ended process for cell")
				log.Debug("moving to next..")


				if resp.Link == nil {
					log.Debug("no target found... hanging up")
					utils.SafeHangup( resp.Channel )
					return
				}
				next := resp.Link
				startProcessingFlow( cl, ctx, flow, resp.Channel, eventVars, next.Target, runner)
		}
	}
}


func ProcessFlow( cl ari.Client, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, eventVars map[string] string, cell *types.Cell) {
	log := utils.GetLogger()
	log.Debug("processing cell type " + cell.Cell.Type)
	runner:=types.Runner{Cancelled: false}
	flow.Runners = append( flow.Runners, &runner )
	startProcessingFlow( cl, ctx, flow, lineChannel, eventVars, cell, &runner)
}