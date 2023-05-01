package mngrs

import (
	//"context"
	"time"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/sirupsen/logrus"

	helpers "github.com/Lineblocs/go-helpers"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)

type PlaybackManager struct {
	ManagerContext *types.Context
	Flow           *types.Flow
}

func NewPlaybackManager(mngrCtx *types.Context, flow *types.Flow) *PlaybackManager {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := PlaybackManager{
		ManagerContext: mngrCtx,
		Flow:           flow}
	return &item
}
func (man *PlaybackManager) StartProcessing() {
	go man.processPlayback()
}

func (man *PlaybackManager) processPlayback() {
	helpers.Log(logrus.DebugLevel, "Creating playback... ")
	cell := man.ManagerContext.Cell
	flow := man.ManagerContext.Flow
	channel := man.ManagerContext.Channel
	model := cell.Model
	data := cell.Model.Data
	next, _ := utils.FindLinkByName(cell.SourceLinks, "source", "Finished")
	playbackType := data["playback_type"].(types.ModelDataStr).Value
	loops := utils.PlaybackLoops(model.Data["number_of_loops"])
	_, _ = utils.FindLinkByName(cell.SourceLinks, "source", "Finished")

	for i := 0; i != loops; i++ {
		switch playbackType {
		case "Say":

			helpers.Log(logrus.DebugLevel, "processing TTS")
			file, err := utils.StartTTS(data["text_to_say"].(types.ModelDataStr).Value,
				data["text_gender"].(types.ModelDataStr).Value,
				data["voice"].(types.ModelDataStr).Value,
				data["text_language"].(types.ModelDataStr).Value)
			if err != nil {
				helpers.Log(logrus.ErrorLevel,"error downloading: " + err.Error())
				man.errorResult()
				return
			}

			man.beginPrompt(file)
			time.Sleep(time.Duration(time.Millisecond * 100))
		case "Play":

			helpers.Log(logrus.DebugLevel, "processing file download")
			file, err := utils.DownloadFile(flow, data["url_audio"].(types.ModelDataStr).Value)

			if err != nil {
				helpers.Log(logrus.ErrorLevel, "error downloading: "+err.Error())
				man.errorResult()
				return
			}

			man.beginPrompt(file)
			time.Sleep(time.Duration(time.Millisecond * 100))
		}
	}
	resp := types.ManagerResponse{
		Channel: channel,
		Link:    next}
	man.ManagerContext.RecvChannel <- &resp
}

func (man *PlaybackManager) errorResult() {
	channel := man.ManagerContext.Channel
	resp := types.ManagerResponse{
		Channel: channel,
		Link:    nil}
	man.ManagerContext.RecvChannel <- &resp
}

func (man *PlaybackManager) beginPrompt(prompt string) {
	channel := man.ManagerContext.Channel
	//cell := man.ManagerContext.Cell
	uri := "sound:" + prompt
	playback, err := channel.Channel.Play(channel.Channel.Key().ID, uri)
	if err != nil {
		helpers.Log(logrus.ErrorLevel, "failed to play join sound, error:"+err.Error())
		return
	}

	//next, _ := utils.FindLinkByName( cell.SourceLinks, "source", "Finished")
	finishedSub := playback.Subscribe(ari.Events.PlaybackFinished)
	defer finishedSub.Cancel()

	helpers.Log(logrus.DebugLevel, "waiting for playback to finish...")
	for {
		select {
		case <-finishedSub.Events():
			helpers.Log(logrus.DebugLevel, "playback finished...")
			/*
				resp := types.ManagerResponse{
					Channel: channel,
					Link: next }
				man.ManagerContext.RecvChannel <- &resp
			*/
			return
		}
	}
}
