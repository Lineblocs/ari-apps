package mngrs
import (
	//"context"
	"github.com/CyCoreSystems/ari/v5"

	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)
type PlaybackManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func NewPlaybackManager(mngrCtx *types.Context, flow *types.Flow) (*PlaybackManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := PlaybackManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *PlaybackManager) StartProcessing() {
	go man.processPlayback()
}

func (man *PlaybackManager) processPlayback() {
	log := man.ManagerContext.Log
	log.Debug( "Creating playback... ")
	cell := man.ManagerContext.Cell
	flow := man.ManagerContext.Flow
	channel := man.ManagerContext.Channel
	model := cell.Model
	data := cell.Model.Data
	next, _ := utils.FindLinkByName( cell.SourceLinks, "source", "Finished")
	playbackType := data["playback_type"].(types.ModelDataStr).Value
	loops := utils.PlaybackLoops( model.Data["number_of_loops"] )
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Finished")

	for i:=0; i != loops; i++ {
		switch ;playbackType { 
			case "Say":


				log.Debug("processing TTS")
				file := "https://lineblocs.s3.ca-central-1.amazonaws.com/media-streams/0c2c67f6-4fcc-11ec-8174-5600039bc38d.wav"
				/*
				file, err := utils.StartTTS(data["text_to_say"].(types.ModelDataStr).Value,
					data["text_gender"].(types.ModelDataStr).Value,
					data["voice"].(types.ModelDataStr).Value,
					data["text_language"].(types.ModelDataStr).Value)
				if err != nil {
					log.Error("error downloading: " + err.Error())
				}
				*/

				man.beginPrompt(file)
			case "Play":

				log.Debug("processing TTS")
				file, err := utils.DownloadFile( flow, data["url_audio"].(types.ModelDataStr).Value)

				if err != nil {
					log.Error("error downloading: " + err.Error())
				}
				man.beginPrompt(file)

		}
	}
	resp := types.ManagerResponse{
		Channel: channel,
		Link: next }
	man.ManagerContext.RecvChannel <- &resp
}
func (man *PlaybackManager) beginPrompt(prompt string) {
	log := man.ManagerContext.Log
	channel := man.ManagerContext.Channel
	cell := man.ManagerContext.Cell
	uri := "sound:" + prompt
	playback, err := channel.Channel.Play(channel.Channel.Key().ID, uri)
	if err != nil {
		log.Error("failed to play join sound", "error", err)
		return
	}

	next, _ := utils.FindLinkByName( cell.SourceLinks, "source", "Finished")
	finishedSub := playback.Subscribe(ari.Events.PlaybackFinished)
	defer finishedSub.Cancel()

	log.Debug("waiting for playback to finish...")
	for {
		select {
		case <-finishedSub.Events():
			log.Debug("playback finished...")
			resp := types.ManagerResponse{
				Channel: channel,
				Link: next }
			man.ManagerContext.RecvChannel <- &resp
			return
		}
	}
}