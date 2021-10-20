package mngrs
import (
	//"context"
	"strings"
	//"github.com/CyCoreSystems/ari/v5"

	"lineblocs.com/processor/types"
	"lineblocs.com/processor/utils"
)
type SwitchManager struct {
	ManagerContext *types.Context
	Flow *types.Flow
}

func NewSwitchManager(mngrCtx *types.Context, flow *types.Flow) (*SwitchManager) {
	//rootCtx, _ := context.WithCancel(context.Background())
	item := SwitchManager{
		ManagerContext:mngrCtx,
		Flow: flow}
	return &item
}
func (man *SwitchManager) StartProcessing() {
	log := man.ManagerContext.Log
	cell := man.ManagerContext.Cell
	flow := man.ManagerContext.Flow
	channel := man.ManagerContext.Channel
	//ctx := man.ManagerContext.Context
	data := cell.Model.Data
	links := cell.Model.Links
	sourceLinks := cell.SourceLinks
	before := data["test_before_interpolations"].ValueStr
	test := data["test"].ValueStr
	var result string
	_, _ = utils.FindLinkByName( cell.SourceLinks, "source", "Finished")

	if strings.HasPrefix(test, "{{") && strings.HasSuffix(test, "}}") {
		result = before
	} else {
		splitted := strings.Split(test, ".")
		if  len( splitted ) > 1 {
			name := splitted[ 0 ]
			variable := strings.Join(splitted[ 1:len( splitted ) - 1 ], ".")
			value, err := utils.LookupCellVariable( flow, name, variable )
			if err != nil {
				log.Debug("cell lookup error: " + err.Error())
			}
			result = value
		}
	}

	var matched *types.ModelLink
	for _, link := range links {
		cond := link.Condition
		condType := link.Type
		value := link.Value
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
			if item.Target.Model.Name == matched.Cell {

				resp := types.ManagerResponse{
					Channel: channel,
					Link: item }
				man.ManagerContext.RecvChannel <- &resp
			}
		}
	}



}
