package mngrs
import (
	//"context"
	//"github.com/CyCoreSystems/ari/v5"

	"lineblocs.com/processor/types"
)
type ConferenceManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func NewConferenceManager(mngrCtx *types.Context, flow *types.Flow) (*ConferenceManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := ConferenceManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *ConferenceManager) StartProcessing() {
	//log := man.ManagerContext.Log
}
