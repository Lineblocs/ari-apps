package mngrs
import (
	//"context"
	//"github.com/CyCoreSystems/ari/v5"
	"github.com/sirupsen/logrus"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/types"
	helpers "github.com/Lineblocs/go-helpers"
	"fmt"
)
type StreamAudioManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func NewStreamAudioManager(mngrCtx *types.Context, flow *types.Flow) (*StreamAudioManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := StreamAudioManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *StreamAudioManager) StartProcessing() {
	//log := man.ManagerContext.Log

	cell := man.ManagerContext.Cell
	amiClient := man.ManagerContext.AMIClient
	data := cell.Model.Data
	next, _ := utils.FindLinkByName(cell.SourceLinks, "source", "Completed")
	fail, _ := utils.FindLinkByName(cell.SourceLinks, "source", "Fail")
	direction := data["direction"].(types.ModelDataStr).Value
	websocketServer := data["websocket_server"].(types.ModelDataStr).Value
	lineChannel := man.ManagerContext.Channel
	options := fmt.Sprintf("D(%s)", direction)
	actionId := "123"
	amiActionParams := map[string]string{
		"Action": "AudioFork",
		"channel": lineChannel.Channel.ID(),
		"WsServer": websocketServer,
		"ActionID": actionId,
		"Command": "StartAudioFork",
		"Options":  options,
	}

	helpers.Log(logrus.InfoLevel, "sending audio stream to: " + websocketServer)
	result, err := amiClient.Action(amiActionParams)
	if err != nil {
		resp := types.ManagerResponse{
			Channel: lineChannel,
			Link:    fail}
		man.ManagerContext.RecvChannel <- &resp
		return
	}
	// If not error, processing result. Response on Action will follow in defined events.
	// You need to catch them in event channel, DefaultHandler or specified HandlerFunction
	fmt.Println(result, err)
	resp := types.ManagerResponse{
		Channel: lineChannel,
		Link:    next}
	man.ManagerContext.RecvChannel <- &resp
}
