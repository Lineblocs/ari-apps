package mngrs
import (
	//"context"
	//"github.com/CyCoreSystems/ari/v5"

	"lineblocs.com/processor/types"
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
}
