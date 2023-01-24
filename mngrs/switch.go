package mngrs

import (
	//"context"
	"strings"
	//"github.com/CyCoreSystems/ari/v5"

	"github.com/sirupsen/logrus"
	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)

type SwitchManager struct {
	ManagerContext *types.Context
	Flow           *types.Flow
}

func NewSwitchManager(mngrCtx *types.Context, flow *types.Flow) *SwitchManager {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := SwitchManager{
		ManagerContext: mngrCtx,
		Flow:           flow}
	return &item
}
func (man *SwitchManager) StartProcessing() {
	go man.startTestForCondition()
}
func (man *SwitchManager) startTestForCondition() {
	cell := man.ManagerContext.Cell
	flow := man.ManagerContext.Flow
	channel := man.ManagerContext.Channel
	//ctx := man.ManagerContext.Context
	data := cell.Model.Data
	links := cell.Model.Links
	sourceLinks := cell.SourceLinks
	before := data["test_before_interpolations"].(types.ModelDataStr).Value
	test := data["test"].(types.ModelDataStr).Value
	var result string
	_, _ = utils.FindLinkByName(cell.SourceLinks, "source", "Finished")

	if strings.HasPrefix(test, "{{") && strings.HasSuffix(test, "}}") {
		result = before
	} else {
		utils.Log(logrus.DebugLevel, "test variable: "+test)
		splitted := strings.Split(test, ".")
		if len(splitted) > 1 {
			name := splitted[0]
			variable := strings.Join(splitted[1:len(splitted)], ".")
			utils.Log(logrus.DebugLevel, "looking UP: "+variable)
			value, err := utils.LookupCellVariable(flow, name, variable)
			if err != nil {
				utils.Log(logrus.DebugLevel, "cell lookup error: "+err.Error())
			}
			result = value
		}
	}
	utils.Log(logrus.DebugLevel, "result is: "+result)

	var matched *types.ModelLink
	for _, link := range links {
		cond := link.Condition
		condType := link.Type
		value := link.Value

		utils.Log(logrus.DebugLevel, "Cond type: "+condType)
		utils.Log(logrus.DebugLevel, "Cond: "+cond)
		utils.Log(logrus.DebugLevel, "Value: "+value)
		if condType == "LINK_CONDITION_MATCHES" {
			if cond == "Equals" && result == value {
				// matched
				matched = link
			} else if cond == "Starts with" && strings.HasPrefix(result, value) {
				// matched
				matched = link
			} else if cond == "Ends with" && strings.HasSuffix(result, value) {
				// matched
				matched = link
			} else if cond == "Matches any" && strings.Contains(result, value) {
				// matched
				matched = link
			}
		}
	}

	if matched != nil {

		for _, item := range sourceLinks {
			utils.Log(logrus.DebugLevel, "comparing 1: "+matched.Cell)
			utils.Log(logrus.DebugLevel, "comparing 2: "+item.Target.Model.Name)
			if item.Target.Model.Name == matched.Cell {
				utils.Log(logrus.DebugLevel, "found match - going to result..")
				resp := types.ManagerResponse{
					Channel: channel,
					Link:    item}
				man.ManagerContext.RecvChannel <- &resp
				return
			}
		}
	}
	resp := types.ManagerResponse{
		Channel: channel,
		Link:    nil}
	man.ManagerContext.RecvChannel <- &resp

}
