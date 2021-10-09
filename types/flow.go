package types
import (
	"fmt"
	//"encoding/json"
	"reflect"
	"github.com/CyCoreSystems/ari/v5"
)
type Vertice struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type CellConnection struct {
	Id string `json:"id"`
	Port string `json:"port"`
}
type GraphCell struct {
	Id string `json:"id"`
	Type string `json:"type"`
	Source CellConnection `json:"source"`
	Target CellConnection `json:"target"`
	Vertices []Vertice `json:"vertices"`
}

type Link struct {
	Link *GraphCell
	Source *Cell
	Target *Cell
}
type Cell struct {
	Cell *GraphCell
	Model *Model
	Channel *LineChannel
	CellChannel *LineChannel
	SourceLinks []*Link
	TargetLinks []*Link
}

type ModelData struct {
	Key string
	ValueStr string
	ValueArr []string
	ValueObj map[string] string
	ValueBool bool
	IsArray bool
	IsObj bool
	IsStr bool
	IsBool bool
}
type Model struct {
	Id string
	Data map[string] ModelData
}

type UnparsedModel struct {
	Id string `json:"id"`
	Data map[string]interface{} `json:"data"`

}
type Graph struct {
	Cells []*GraphCell `json:"cells"`
}
type FlowVars struct {
	Graph Graph `json:"graph"`
	Models []UnparsedModel `json:"models"`
}

type FlowDIDData struct {
	//FlowJson FlowVars `json:"flow_json"`
	WorkspaceId int `json:"workspace_id"`
	WorkspaceName string `json:"workspace_name"`
	CreatorId int `json:"creator_id"`
	FlowJson string `json:"flow_json"`
	Plan string `json:"plan"`
}
func findCellInFlow(id string, flow *Flow, channel *LineChannel) (*Cell) {

	var cellToFind *GraphCell
	for _, cell := range flow.Vars.Graph.Cells {
		if cell.Id == id {
			cellToFind = cell
		}
	}
	if cellToFind == nil {
		// could not find
	}
	cell := Cell{ Channel: channel, Cell: cellToFind }
	if cellToFind.Type == "devs.DialModel" || cellToFind.Type == "devs.BridgeModel" || cellToFind.Type == "devs.ConferenceModel" {
		// empty holder channel
		cell.CellChannel = &LineChannel{}
	}
	return &cell
}
	
func createCellData(cell *Cell, flow *Flow, channel *LineChannel) {
	var model Model = Model{
		Id: "",
		Data: make(map[string] ModelData)}
	sourceLinks := make( []*Link, 0 )
	targetLinks := make( []*Link, 0 )
	for _, item := range flow.Vars.Models {
		if (item.Id == cell.Cell.Id) {
			//unparsedModel := item
			var modelData map[string]interface{}
			modelData = item.Data
			//json.Unmarshal([]byte(unparsedModel.Data), &modelData)

			for key, v := range modelData {
				item := ModelData{}
 				typeOfValue := fmt.Sprintf("%s", reflect.TypeOf(v))

				fmt.Printf("parsing type %s\r\n", typeOfValue)
				switch ; typeOfValue {
					case "[]string":
						// it's an array

						item.ValueArr = v.([]string)
						//item.ValueArr = v
						item.IsArray = true

					case "map[string]string":
						// it's an object
						fmt.Println("converting obj")
						item.ValueObj = v.(map[string]string)
						//item.ValueObj = v
						item.IsObj = true
					case "string":
						// it's something else
						item.ValueStr = v.(string)
						item.IsStr = true
					case "boolean":
						// it's something else
						item.ValueBool = v.(bool)
						item.IsBool = true
					}
					model.Data[key] = item

			}
		}
	}	

	cell.Model = &model

	for _, item := range flow.Vars.Graph.Cells {
		if item.Type == "devs.FlowLink" {
			fmt.Printf("processing link %s", item.Type)
			if item.Source.Id == cell.Cell.Id {
				destCell := addCellToFlow( item.Target.Id, flow, channel )
				link := &Link{
					Link: item,
					Source: cell,
					Target: destCell }
				sourceLinks = append( sourceLinks, link )
			} else if item.Target.Id == cell.Cell.Id {
				srcCell := addCellToFlow( item.Target.Id, flow, channel )
				link := &Link{
					Link: item,
					Source: srcCell,
					Target: cell }
				targetLinks = append( targetLinks, link )
			}

		}
	}
	cell.SourceLinks = sourceLinks
	cell.TargetLinks = targetLinks
}
func addCellToFlow(id string, flow *Flow, channel *LineChannel) (*Cell) {

	for _, cell := range flow.Cells {
		if cell.Cell.Id == id {
			return cell
		}
	}

	cellInFlow := findCellInFlow(id, flow, channel)
	flow.Cells = append(flow.Cells, cellInFlow)
	createCellData(cellInFlow, flow, channel)
	return cellInFlow
}
func NewFlow(user *User, vars *FlowVars, channel *LineChannel, client ari.Client) (*Flow) {
	flow := &Flow{User: user, Vars: vars, Runners: make([]*Runner, 0)}
	fmt.Printf("number of cells %d\r\n", len(flow.Vars.Graph.Cells))
	// create cells from flow.Vars
	for _, cell := range flow.Vars.Graph.Cells {
		fmt.Printf("processing %s\r\n", cell.Type)
		// creating a cell	
		if cell != nil {
			if cell.Type != "devs.FlowLink" {
				addCellToFlow( cell.Id, flow, channel )
			}
		}
	}
	return flow
}


type Flow struct {
	User *User
	Exten string
	CallerId string
	Channel *LineChannel
	RootCall *Call
	Cells []*Cell
	Models []*Model
	Runners []*Runner
	Vars *FlowVars
}

type Runner struct {
	Cancelled bool
}

