package mngrs
import (
	//"context"
	//"github.com/CyCoreSystems/ari/v5"

	"lineblocs.com/processor/types"
)
type WaitManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func NewWaitManager(mngrCtx *types.Context, flow *types.Flow) (*WaitManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := WaitManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *WaitManager) StartProcessing() {
	log := man.ManagerContext.Log
	log.Debug("starting wait...")
	//man.ManagerContext.RecvChannel <- *item
}
