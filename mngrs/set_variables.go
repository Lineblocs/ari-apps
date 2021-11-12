package mngrs
import (
	//"context"
	//"github.com/CyCoreSystems/ari/v5"

	"lineblocs.com/processor/types"
)
type SetVariablesManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func NewSetVariablesManager(mngrCtx *types.Context, flow *types.Flow) (*SetVariablesManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := SetVariablesManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *SetVariablesManager) StartProcessing() {
	//log := man.ManagerContext.Log
}
