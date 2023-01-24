package mngrs

import (
	//"github.com/CyCoreSystems/ari/v5"
	//clientcmd "k8s.io/client-go/1.5/tools/clientcmd"
	//"k8s.io/client-go/kubernetes"
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/sirupsen/logrus"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"

	"google.golang.org/grpc"
	"lineblocs.com/processor/router"
)

func (man *MacroManager) startGRPCAndRunMacro(macro *types.WorkspaceMacro, params map[string]string) error {
	ctx := man.ManagerContext
	user := ctx.Flow.User
	channel := ctx.Channel
	flow := ctx.Flow
	cell := ctx.Cell
	var conn *grpc.ClientConn

	// get the service URL for the user
	// <service-name>.<namespace>.svc.cluster.local:<service-port>
	name := "voip-users"
	port := "10000"
	svcUri := fmt.Sprintf("%s.%s.svc.cluster.local:%s", user.WorkspaceName, name, port)
	conn, err := grpc.Dial(svcUri, grpc.WithInsecure())
	if err != nil {
		utils.Log(logrus.ErrorLevel, "did not connect: "+err.Error())
		return err
	}
	defer conn.Close()

	c := router.NewLineblocsWorspaceSvcClient(conn)
	params["channel_id"] = channel.Channel.ID()
	params["flow_id"] = strconv.Itoa(flow.FlowId)
	params["cell_id"] = cell.Cell.Id
	params["cell_name"] = cell.Cell.Name
	eventCtx := router.EventContext{
		Name:  macro.Title,
		Event: params}
	response, err := c.CallMacro(context.Background(), &eventCtx)
	if err != nil {
		utils.Log(logrus.ErrorLevel, "Error when calling CallMacro: "+err.Error())
		return err
	}
	if response.Error {
		utils.Log(logrus.InfoLevel, "macro resulted in error: "+response.Msg)
		return errors.New(response.Msg)
	}
	utils.Log(logrus.InfoLevel, "Response from server: "+response.Result)
	return nil
}

type MacroManager struct {
	ManagerContext *types.Context
	Flow           *types.Flow
}

func NewMacroManager(mngrCtx *types.Context, flow *types.Flow) *MacroManager {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := MacroManager{
		ManagerContext: mngrCtx,
		Flow:           flow}
	return &item
}

func (man *MacroManager) executeMacro() {
	cell := man.ManagerContext.Cell
	channel := man.ManagerContext.Channel
	flow := man.ManagerContext.Flow
	model := cell.Model
	utils.Log(logrus.DebugLevel, "running macro script..")

	function := model.Data["function"].(types.ModelDataStr).Value
	params := model.Data["params"].(types.ModelDataObj).Value

	completed, _ := utils.FindLinkByName(cell.SourceLinks, "source", "Completed")
	errorLink, _ := utils.FindLinkByName(cell.SourceLinks, "source", "Error")

	var foundFn *types.WorkspaceMacro

	// find the code
	for _, macro := range flow.WorkspaceFns {
		if macro.Title == function {
			foundFn = macro
		}
	}

	if foundFn == nil {
		utils.Log(logrus.DebugLevel, "could not find macro function...")
		resp := types.ManagerResponse{
			Channel: channel,
			Link:    errorLink}
		man.ManagerContext.RecvChannel <- &resp
		return
	}

	err := man.startGRPCAndRunMacro(foundFn, params)
	//sEnc := b64.StdEncoding.EncodeToString([]byte(foundFn.CompiledCode))
	//err := man.initializeK8sAndExecute(sEnc)

	if err != nil {
		utils.Log(logrus.ErrorLevel, "error occured: "+err.Error())
		resp := types.ManagerResponse{
			Channel: channel,
			Link:    errorLink}
		man.ManagerContext.RecvChannel <- &resp
		return
	}
	resp := types.ManagerResponse{
		Channel: channel,
		Link:    completed}
	man.ManagerContext.RecvChannel <- &resp
}
func (man *MacroManager) StartProcessing() {
	go man.executeMacro()
}
