package mngrs

import (
	"context"
	"strconv"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/sirupsen/logrus"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)

type BaseManager interface {
	StartProcessing()
}

func startProcessingFlow(cl ari.Client, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, eventVars map[string]string, cell *types.Cell, runner *types.Runner) {
	utils.Log(logrus.DebugLevel, "processing cell type "+cell.Cell.Type)
	if runner.Cancelled {
		utils.Log(logrus.DebugLevel, "flow runner was cancelled - exiting")
		return
	}
	utils.Log(logrus.DebugLevel, "source link count: "+strconv.Itoa(len(cell.SourceLinks)))
	utils.Log(logrus.DebugLevel, "target link count: "+strconv.Itoa(len(cell.TargetLinks)))

	manRecvChannel := make(chan *types.ManagerResponse)
	lineCtx := types.NewContext(
		cl,
		ctx,
		manRecvChannel,
		flow,
		cell,
		runner,
		lineChannel)
	// execute it
	var mngr BaseManager
	switch cell.Cell.Type {
	case "devs.LaunchModel":
		for _, link := range cell.SourceLinks {
			go startProcessingFlow(cl, ctx, flow, lineChannel, eventVars, link.Target, runner)
		}
		return
	case "devs.SwitchModel":
		mngr = NewSwitchManager(lineCtx, flow)
	case "devs.BridgeModel":
		mngr = NewBridgeManager(lineCtx, flow)
	case "devs.PlaybackModel":
		mngr = NewPlaybackManager(lineCtx, flow)
	case "devs.ProcessInputModel":
		mngr = NewInputManager(lineCtx, flow)
	case "devs.DialModel":
		mngr = NewDialManager(lineCtx, flow)
	case "devs.SetVariablesModel":
		mngr = NewSetVariablesManager(lineCtx, flow)
	case "devs.WaitModel":
		mngr = NewWaitManager(lineCtx, flow)
	case "devs.SendDigitsModel":
		mngr = NewSendDigitsManager(lineCtx, flow)
	case "devs.MacroModel":
		mngr = NewMacroManager(lineCtx, flow)
	case "devs.ConferenceModel":
		mngr = NewConferenceManager(lineCtx, flow)
	default:
		utils.Log(logrus.InfoLevel, "unknown type of cell..")
		return

	}
	mngr.StartProcessing()

	utils.Log(logrus.DebugLevel, "waiting to receive from channel...")
	for {
		select {
		case resp, ok := <-manRecvChannel:
			if !ok {
				utils.Log(logrus.DebugLevel, "error receiving result from cell..")
				return
			}
			utils.Log(logrus.DebugLevel, "ended process for cell")
			utils.Log(logrus.DebugLevel, "moving to next..")

			if resp.Link == nil {
				utils.Log(logrus.DebugLevel, "no target found... hanging up")
				utils.SafeHangup(resp.Channel)
				return
			}
			next := resp.Link
			defer startProcessingFlow(cl, ctx, flow, resp.Channel, eventVars, next.Target, runner)
			return
		}
	}
}

func ProcessFlow(cl ari.Client, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, eventVars map[string]string, cell *types.Cell) {
	utils.Log(logrus.DebugLevel, "processing cell type "+cell.Cell.Type)
	runner := types.Runner{Cancelled: false}
	flow.Runners = append(flow.Runners, &runner)
	startProcessingFlow(cl, ctx, flow, lineChannel, eventVars, cell, &runner)
}
