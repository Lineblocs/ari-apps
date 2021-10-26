package mngrs
import (
	//"context"
	//"github.com/CyCoreSystems/ari/v5"
	"fmt"
	"lineblocs.com/processor/utils"
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


func (man *MacroManager) executeMacro() {
	log := man.ManagerContext.Log
	cell := man.ManagerContext.Cell
	channel := man.ManagerContext.Channel
	flow := man.ManagerContext.Flow
	model := cell.Model
	log.Debug("running macro script..");

	function := model.Data["function"].ValueStr

	completed, _ := utils.FindLinkByName( cell.SourceLinks, "source", "Completed")
	errorLink, _ := utils.FindLinkByName( cell.SourceLinks, "source", "Error")

	var foundFn *types.WorkspaceMacro
	// find the code
	for _, macro := range flow.WorkspaceFns {
		if macro.Title ==  function {
			foundFn = macro
		}
	}
	if foundFn == nil {
		log.Debug("could not find macro function...")
		resp := types.ManagerResponse{
			Channel: channel,
			Link: errorLink }
		man.ManagerContext.RecvChannel <- &resp
		return
	}

	err := utils.RunScriptInContext(foundFn.CompiledCode)
	if err != nil {
		fmt.Println( err.Error ());

		resp := types.ManagerResponse{
			Channel: channel,
			Link: errorLink }
		man.ManagerContext.RecvChannel <- &resp
		return

	}
	resp := types.ManagerResponse{
		Channel: channel,
		Link: completed }
	man.ManagerContext.RecvChannel <- &resp
}
func (man *MacroManager) StartProcessing() {
	go man.executeMacro();
}
