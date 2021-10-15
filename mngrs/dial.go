package mngrs
import (
	//"context"
	"lineblocs.com/processor/types"
)
type DialManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func NewDialManager(mngrCtx *types.Context, flow *types.Flow) (*DialManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := DialManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *DialManager) StartProcessing() {
	log := man.ManagerContext.Log
	log.Debug( "Creating bridge... ")
}

