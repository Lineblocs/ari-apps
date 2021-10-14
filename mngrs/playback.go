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
	log := man.ManagerContext.Log
	log.Debug( "Creating playback... ")
	cell := man.ManagerContext.Cell
	flow := man.ManagerContext.Flow
	data := cell.Model.Data
	playbackType := data["playback_type"].ValueStr
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Finished")

	if playbackType == "Say" {


		log.Debug("processing TTS")
		file, err := utils.StartTTS( flow, 
			data["text_to_say"].ValueStr,
			data["text_gender"].ValueStr,
			data["voice"].ValueStr,
			data["text_language"].ValueStr,
		)
		if err != nil {
			log.Error("error downloading: " + err.Error())
		}

		man.beginPrompt(file)
	} else if playbackType == "Play" {

		log.Debug("processing TTS")
		file, err := utils.DownloadFile( flow, data["url_audio"].ValueStr )

		if err != nil {
			log.Error("error downloading: " + err.Error())
		}
		man.beginPrompt(file)

	}

}

func (man *PlaybackManager) beginPrompt(prompt string) {
	log := man.ManagerContext.Log
	log.Debug( "Creating playback... ")
	channel := man.ManagerContext.Channel
	uri := "sound:" + prompt
	playback, err := channel.Channel.Play(channel.Channel.Key().ID, uri)
	if err != nil {
		log.Error("failed to play join sound", "error", err)
		return
	}
	finishedSub := playback.Subscribe(ari.Events.PlaybackFinished)
	defer finishedSub.Cancel()

	for {
		select {
		case <-finishedSub.Events():
			log.Debug("playback finished...")
			return
		default:
			log.Debug("no response received..")
			return
		}
	}
}