package mngrs
import (
	//"context"
	"strconv"
	"time"
	"sync"
	"github.com/CyCoreSystems/ari/v5"

	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)
type InputManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func NewInputManager(mngrCtx *types.Context, flow *types.Flow) (*InputManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := InputManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *InputManager) StartProcessing() {
	log := man.ManagerContext.Log
	log.Debug( "Creating playback for INPUT... ")
	cell := man.ManagerContext.Cell
	flow := man.ManagerContext.Flow
	data := cell.Model.Data
	playbackType := data["playback_type"].ValueStr
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Finished")
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Finished")

	stopTimeout, err := strconv.ParseFloat( data["stop_timeout"].ValueStr, 64 )

	if err != nil {
		log.Debug("error parsing stop timeout. value was:  " + data["stop_timeout"].ValueStr)
		return
	}

	maxDigits, err := strconv.Atoi( data["max_digits"].ValueStr )

	stopGatherOnKeypress := data["stop_gather_on_keypress"].ValueBool
	keypressKeyStop := data["keypress_key_stop"].ValueStr
	stopChannel := make( chan bool, 1 )
 	wg1 := new(sync.WaitGroup)
	wg1.Add(1)
	go man.attachDtmfListeners(stopTimeout, maxDigits, stopGatherOnKeypress, keypressKeyStop, wg1, stopChannel)
	wg1.Wait()

	if err != nil {
		log.Debug("error parsing max digits. value was:  " + data["max_digits"].ValueStr)
		return
	}

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

		go man.beginPrompt(file, stopChannel)
	} else if playbackType == "Play" {

		log.Debug("processing TTS")
		file, err := utils.DownloadFile( flow, data["url_audio"].ValueStr )

		if err != nil {
			log.Error("error downloading: " + err.Error())
		}
		go man.beginPrompt(file, stopChannel)

	}
}

func (man *InputManager) attachDtmfListeners(stopTimeout float64, maxDigits int, stopGatherOnKeypress bool, keypressKeyStop string, wg *sync.WaitGroup, stopChannel chan<- bool) {
	log := man.ManagerContext.Log

	channel := man.ManagerContext.Channel
	ctx := man.ManagerContext.Context
	log.Debug( "listening for DTMF.." )
	dtmfSub := channel.Channel.Subscribe(ari.Events.ChannelDtmfReceived)
	defer dtmfSub.Cancel()
	var timeLastDtmfWasReceived *time.Time
	timeLastDtmfWasReceived= nil
	collectedDtmf := ""

	wg.Done()
	for {

		select {
			case <-ctx.Done():
				return
			case e, ok := <-dtmfSub.Events():

				if !ok {
					log.Debug("error fetching event")
					return
				}


				v := e.(*ari.ChannelDtmfReceived)
				digit := v.Digit
				log.Debug("input received DTMF: " + digit)

				// stop due to key pressed
				if stopGatherOnKeypress && digit == keypressKeyStop {
					stopChannel <- true
					man.finishProcessingDTMF(collectedDtmf)
					return
				}

				collectedDtmf += digit
				if timeLastDtmfWasReceived != nil {
					elapsed := time.Since(*timeLastDtmfWasReceived).Seconds()

					if elapsed > stopTimeout {
						// time was elapsed
						stopChannel <- true
						man.finishProcessingDTMF(collectedDtmf)
						return
					}
				}

				if len( collectedDtmf ) >= maxDigits {
					// max digits
					stopChannel <- true
					man.finishProcessingDTMF(collectedDtmf)
					return
				}

				now := time.Now()
				timeLastDtmfWasReceived =&now
		}
	}
}


func (man *InputManager) beginPrompt(prompt string, stopChannel <-chan bool) {
	log := man.ManagerContext.Log
	channel := man.ManagerContext.Channel
	uri := "sound:" + prompt
	playback, err := channel.Channel.Play(channel.Channel.Key().ID, uri)
	if err != nil {
		log.Error("failed to play join sound", "error", err)
		return
	}
	finishedSub := playback.Subscribe(ari.Events.PlaybackFinished)
	defer finishedSub.Cancel()
	log.Debug("PLAYBACK started...");

	for {
		select {
		case <-finishedSub.Events():
			log.Debug("playback finished...")
			return

		case <-stopChannel:
			log.Debug("requested playback stop..");
			err := playback.Stop()
			if err != nil {
				log.Debug("error occured: " + err.Error());
			}
			return
		}
	}
}

func (man *InputManager) finishProcessingDTMF(result string) {
 	ctx := man.ManagerContext
	log := man.ManagerContext.Log

	cell := man.ManagerContext.Cell

	log.Debug("finish processing DTMF...")
	cell.EventVars["digits"] = result
	digits, _ := utils.FindLinkByName( cell.SourceLinks, "source", "Digits Received")
	resp := types.ManagerResponse{
			Channel: ctx.Channel, Link: digits }
	man.ManagerContext.RecvChannel <- &resp
}