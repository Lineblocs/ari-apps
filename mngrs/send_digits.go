package mngrs
import (
	//"context"
	//"github.com/CyCoreSystems/ari/v5"
	"github.com/sirupsen/logrus"
	"lineblocs.com/processor/utils"
	"lineblocs.com/processor/types"
	helpers "github.com/Lineblocs/go-helpers"
)
type SendDigitsManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func NewSendDigitsManager(mngrCtx *types.Context, flow *types.Flow) (*SendDigitsManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := SendDigitsManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *SendDigitsManager) StartProcessing() {
	//log := man.ManagerContext.Log

	cell := man.ManagerContext.Cell
	//model := cell.Model
	data := cell.Model.Data
	next, _ := utils.FindLinkByName(cell.SourceLinks, "source", "Finished")
	keys := data["text"].(types.ModelDataStr).Value
	lineChannel := man.ManagerContext.Channel
	//dtmfOpts := &ari.DTMFOptions{}

	for i := 0; i < len(keys); i++ {
    	key:= string( keys[i] )
		helpers.Log(logrus.DebugLevel, "sending DTMF " + key)
		//lineChannel.Channel.SendDTMF(key, dtmfOpts)
		lineChannel.Channel.SendDTMF(key, nil)
 	}
	resp := types.ManagerResponse{
		Channel: lineChannel,
		Link:    next}
	man.ManagerContext.RecvChannel <- &resp
}
