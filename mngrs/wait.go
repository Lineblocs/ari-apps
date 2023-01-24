package mngrs

import (
	//"context"
	"time"
	//"github.com/CyCoreSystems/ari/v5"
	"strconv"

	"github.com/sirupsen/logrus"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)

type WaitManager struct {
	ManagerContext *types.Context
	Flow           *types.Flow
}

func NewWaitManager(mngrCtx *types.Context, flow *types.Flow) *WaitManager {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := WaitManager{
		ManagerContext: mngrCtx,
		Flow:           flow}
	return &item
}
func (man *WaitManager) StartProcessing() {
	utils.Log(logrus.DebugLevel, "starting WAIT...")
	//man.ManagerContext.RecvChannel <- *item

	ctx := man.ManagerContext
	cell := ctx.Cell
	channel := ctx.Channel
	model := cell.Model
	completed, _ := utils.FindLinkByName(cell.SourceLinks, "source", "Completed")

	val, err := strconv.Atoi(model.Data["wait_seconds"].(types.ModelDataStr).Value)
	if err != nil {
		utils.Log(logrus.DebugLevel, "could not parse wait timeout of: "+model.Data["wait_seconds"].(types.ModelDataStr).Value)
		man.ManagerContext.RecvChannel <- nil
		return
	}

	time.Sleep(time.Duration(val) * time.Second)

	resp := types.ManagerResponse{
		Channel: channel,
		Link:    completed}
	man.ManagerContext.RecvChannel <- &resp
}
