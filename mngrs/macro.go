package mngrs
import (
	//"context"
	//"github.com/CyCoreSystems/ari/v5"

	"lineblocs.com/processor/types"
)
type MacroManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func NewMacroManager(mngrCtx *types.Context, flow *types.Flow) (*MacroManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := MacroManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *MacroManager) StartProcessing() {
	//log := man.ManagerContext.Log
}
