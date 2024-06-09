package mngrs

import (
	"context"
	"strconv"

	"github.com/CyCoreSystems/ari/v5"
	helpers "github.com/Lineblocs/go-helpers"
	"github.com/sirupsen/logrus"
	"github.com/ivahaev/amigo"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)

type BaseManager interface {
	StartProcessing()
}

func startProcessingFlow(cl ari.Client, amiClient *amigo.Amigo, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, eventVars map[string]string, cell *types.Cell, runner *types.Runner) {
	helpers.Log(logrus.DebugLevel, "startProcessingFlow processing cell type "+cell.Cell.Type)
	if runner.Cancelled {
		helpers.Log(logrus.DebugLevel, "flow runner was cancelled - exiting")
		return
	}
	helpers.Log(logrus.DebugLevel, "source link count: "+strconv.Itoa(len(cell.SourceLinks)))
	helpers.Log(logrus.DebugLevel, "target link count: "+strconv.Itoa(len(cell.TargetLinks)))

	manRecvChannel := make(chan *types.ManagerResponse)
	lineCtx := types.NewContext(
		cl,
		amiClient,
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
			go startProcessingFlow(cl, amiClient, ctx, flow, lineChannel, eventVars, link.Target, runner)
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
	case "devs.StreamAudioModel":
		mngr = NewStreamAudioManager(lineCtx, flow)
	case "devs.SendDigitsModel":
		mngr = NewSendDigitsManager(lineCtx, flow)
	case "devs.MacroModel":
		mngr = NewMacroManager(lineCtx, flow)
	case "devs.ConferenceModel":
		mngr = NewConferenceManager(lineCtx, flow)
	default:
		helpers.Log(logrus.InfoLevel, "unknown type of cell..")
		return

	}
	mngr.StartProcessing()

	helpers.Log(logrus.DebugLevel, "waiting to receive from channel...")
	for {
		select {
		case resp, ok := <-manRecvChannel:
			if !ok {
				helpers.Log(logrus.DebugLevel, "error receiving result from cell..")
				return
			}
			helpers.Log(logrus.DebugLevel, "ended process for cell")
			helpers.Log(logrus.DebugLevel, "moving to next..")

			if resp.Link == nil {
				helpers.Log(logrus.DebugLevel, "no target found... hanging up")
				resp.Channel.SafeHangup()
				return
			}
			next := resp.Link
			defer startProcessingFlow(cl, amiClient, ctx, flow, resp.Channel, eventVars, next.Target, runner)
			return
		}
	}
}

func ProcessFlow(cl ari.Client, ctx context.Context, flow *types.Flow, lineChannel *types.LineChannel, eventVars map[string]string, cell *types.Cell) {
	var amiClient *amigo.Amigo
	var err error

	runner := types.Runner{Cancelled: false}
	flow.Runners = append(flow.Runners, &runner)
	needsAMI := false

	if needsAMI {
		// TODO: fix async code around AMI client instantiation
		amiClient, err = utils.CreateAMIClient()
		if err != nil {
			helpers.Log(logrus.ErrorLevel, "ProcessFlow error: could not create AMI client")
			return
		}
	}

	startProcessingFlow(cl, amiClient, ctx, flow, lineChannel, eventVars, cell, &runner)
}
