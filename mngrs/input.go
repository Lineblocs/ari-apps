package mngrs

import (
	//"context"
	"strconv"
	"sync"
	"time"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/sirupsen/logrus"

	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)

type InputManager struct {
	ManagerContext *types.Context
	Flow           *types.Flow
}

func NewInputManager(mngrCtx *types.Context, flow *types.Flow) *InputManager {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := InputManager{
		ManagerContext: mngrCtx,
		Flow:           flow}
	return &item
}
func (man *InputManager) StartProcessing() {
	utils.Log(logrus.DebugLevel, "Creating playback for INPUT... ")
	cell := man.ManagerContext.Cell
	flow := man.ManagerContext.Flow
	data := cell.Model.Data
	playbackType := data["playback_type"].(types.ModelDataStr).Value
	_, _ = utils.FindLinkByName(cell.SourceLinks, "source", "Finished")
	_, _ = utils.FindLinkByName(cell.SourceLinks, "source", "Finished")

	stopTimeout, err := strconv.ParseFloat(data["stop_timeout"].(types.ModelDataStr).Value, 64)

	if err != nil {
		utils.Log(logrus.DebugLevel, "error parsing stop timeout. value was:  "+data["stop_timeout"].(types.ModelDataStr).Value)
		return
	}

	maxDigits, err := strconv.Atoi(data["max_digits"].(types.ModelDataStr).Value)

	stopGatherOnKeypress := data["stop_gather_on_keypress"].(types.ModelDataBool).Value
	keypressKeyStop := data["keypress_key_stop"].(types.ModelDataStr).Value
	stopChannel := make(chan bool, 1)
	wg1 := new(sync.WaitGroup)
	wg1.Add(1)
	go man.attachDtmfListeners(stopTimeout, maxDigits, stopGatherOnKeypress, keypressKeyStop, wg1, stopChannel)
	wg1.Wait()

	if err != nil {
		utils.Log(logrus.DebugLevel, "error parsing max digits. value was:  "+data["max_digits"].(types.ModelDataStr).Value)
		return
	}

	switch playbackType {
	case "Say":

		utils.Log(logrus.DebugLevel, "processing TTS")
		file, err := utils.StartTTS(data["text_to_say"].(types.ModelDataStr).Value,
			data["text_gender"].(types.ModelDataStr).Value,
			data["voice"].(types.ModelDataStr).Value,
			data["text_language"].(types.ModelDataStr).Value,
		)
		if err != nil {
			utils.Log(logrus.ErrorLevel, "error downloading: "+err.Error())
		}

		go man.beginPrompt(file, stopChannel)
	case "Play":

		utils.Log(logrus.DebugLevel, "processing TTS")
		file, err := utils.DownloadFile(flow, data["url_audio"].(types.ModelDataStr).Value)

		if err != nil {
			utils.Log(logrus.ErrorLevel, "error downloading: "+err.Error())
		}
		go man.beginPrompt(file, stopChannel)

	}
}

func (man *InputManager) attachDtmfListeners(stopTimeout float64, maxDigits int, stopGatherOnKeypress bool, keypressKeyStop string, wg *sync.WaitGroup, stopChannel chan<- bool) {
	channel := man.ManagerContext.Channel
	ctx := man.ManagerContext.Context
	utils.Log(logrus.DebugLevel, "listening for DTMF..")
	dtmfSub := channel.Channel.Subscribe(ari.Events.ChannelDtmfReceived)
	defer dtmfSub.Cancel()
	var timeLastDtmfWasReceived *time.Time
	timeLastDtmfWasReceived = nil
	collectedDtmf := ""

	wg.Done()
	for {

		select {
		case <-ctx.Done():
			return
		case e, ok := <-dtmfSub.Events():

			if !ok {
				utils.Log(logrus.DebugLevel, "error fetching event")
				return
			}

			v := e.(*ari.ChannelDtmfReceived)
			digit := v.Digit
			utils.Log(logrus.DebugLevel, "input received DTMF: "+digit)

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

			if len(collectedDtmf) >= maxDigits {
				// max digits
				stopChannel <- true
				man.finishProcessingDTMF(collectedDtmf)
				return
			}

			now := time.Now()
			timeLastDtmfWasReceived = &now
		}
	}
}

func (man *InputManager) beginPrompt(prompt string, stopChannel <-chan bool) {
	channel := man.ManagerContext.Channel
	uri := "sound:" + prompt
	playback, err := channel.Channel.Play(channel.Channel.Key().ID, uri)
	if err != nil {
		utils.Log(logrus.ErrorLevel, "failed to play join sound, error:"+err.Error())
		return
	}
	finishedSub := playback.Subscribe(ari.Events.PlaybackFinished)
	defer finishedSub.Cancel()
	utils.Log(logrus.DebugLevel, "PLAYBACK started...")

	for {
		select {
		case <-finishedSub.Events():
			utils.Log(logrus.DebugLevel, "playback finished...")
			return

		case <-stopChannel:
			utils.Log(logrus.DebugLevel, "requested playback stop..")
			err := playback.Stop()
			if err != nil {
				utils.Log(logrus.DebugLevel, "error occured: "+err.Error())
			}
			return
		}
	}
}

func (man *InputManager) finishProcessingDTMF(result string) {
	ctx := man.ManagerContext

	cell := man.ManagerContext.Cell

	utils.Log(logrus.DebugLevel, "finish processing DTMF...")
	cell.EventVars["digits"] = result
	digits, _ := utils.FindLinkByName(cell.SourceLinks, "source", "Digits Received")
	resp := types.ManagerResponse{
		Channel: ctx.Channel, Link: digits}
	man.ManagerContext.RecvChannel <- &resp
}
