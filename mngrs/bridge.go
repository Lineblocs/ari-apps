package mngrs
import (
	"lineblocs.com/processor/types"
)

type BridgeManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}
func startSimpleCall() {

}
func NewBridgeManager(mngrCtx *types.Context, flow *types.Flow) (*BridgeManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := BridgeManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *BridgeManager) StartProcessing() {
	for {
		select {
			case <-man.ManagerContext.Context.Done():
				return
		}
	}
}