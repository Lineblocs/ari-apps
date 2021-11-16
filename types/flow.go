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
	Name string `json:"name"`
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
	EventVars map[string]string
	AttachedCall *Call
}

type ModelData interface {
}

type ModelDataStr struct {
	Value string
}
type ModelDataBool struct {
	Value bool
}
type ModelDataArr struct {
	Value []string
}
type ModelDataObj struct {
	Value map[string] string
}
type ModelLink struct {
	Type string `json:"type"`
	Condition string `json:"condition"`
	Value string `json:"value"`
	Cell string `json:"cell"`
}
type Model struct {
	Id string
	Name string
	Data map[string] ModelData
	Links []*ModelLink `json:"links"`
}

type UnparsedModel struct {
	Id string `json:"id"`
	Name string `json:"name"`
	Data map[string]interface{} `json:"data"`
	Links []*ModelLink `json:"links"`

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
	FlowId int `json:"flow_id"`
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
	cell := Cell{ Channel: channel, Cell: cellToFind, EventVars: make( map[string]string ) }
	if cellToFind.Type == "devs.DialModel" || cellToFind.Type == "devs.BridgeModel" || cellToFind.Type == "devs.ConferenceModel" {
		// empty holder channel
		cell.CellChannel = &LineChannel{}
	}
	return &cell
}
	
func createCellData(cell *Cell, flow *Flow, channel *LineChannel) {
	var model Model = Model{
		Id: "",
		Data: make(map[string] ModelData) }
	sourceLinks := make( []*Link, 0 )
	targetLinks := make( []*Link, 0 )
	for _, item := range flow.Vars.Models {
		if (item.Id == cell.Cell.Id) {
			//unparsedModel := item
			var modelData map[string]interface{}
			modelData = item.Data
			model.Name =  item.Name
			model.Links =  item.Links
			//json.Unmarshal([]byte(unparsedModel.Data), &modelData)

			for key, v := range modelData {
				//var item ModelData
				//item = ModelData{}
 				typeOfValue := fmt.Sprintf("%s", reflect.TypeOf(v))

				fmt.Printf("parsing type %s\r\n", typeOfValue)
					fmt.Printf("setting key: %s\r\n", key)
				switch ; typeOfValue {
					case "[]string":
						// it's an array
						value := v.([]string)
						model.Data[key]= ModelDataArr{Value: value}
						//item.ValueArr = v
						//item.IsArray = true

					case "map[string]string":
						// it's an object
						fmt.Println("converting obj")
						value := v.(map[string]string)
						model.Data[key]=ModelDataObj{Value: value}
						//item.ValueObj = v
						//item.IsObj = true
					case "string":
						// it's something else
						//item.ValueStr = v.(string)
						//item.IsStr = true
						value := v.(string)
						model.Data[key]=ModelDataStr{Value: value}
					case "boolean":
						// it's something else
						//item.ValueBool = v.(bool)
						//item.IsBool = true
						value := v.(bool)
						model.Data[key]=ModelDataBool{Value: value}
					}

			}
		}
	}	

	cell.Model = &model

	for _, item := range flow.Vars.Graph.Cells {
		if item.Type == "devs.FlowLink" {
			fmt.Printf("createCellData processing link %s\r\n", item.Type)
			if item.Source.Id == cell.Cell.Id {
				fmt.Printf("createCellData adding target link %s\r\n", item.Target.Id)
				destCell := addCellToFlow( item.Target.Id, flow, channel )
				link := &Link{
					Link: item,
					Source: cell,
					Target: destCell }
				sourceLinks = append( sourceLinks, link )
			} else if item.Target.Id == cell.Cell.Id {
				fmt.Printf("createCellData adding source link %s\r\n", item.Target.Id)
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

	fmt.Printf("adding cell %s", cellInFlow.Cell.Id)
	flow.Cells = append(flow.Cells, cellInFlow)
	createCellData(cellInFlow, flow, channel)
	return cellInFlow
}
func NewFlow(id int, user *User, vars *FlowVars, channel *LineChannel, fns []*WorkspaceMacro, client ari.Client) (*Flow) {
	flow := &Flow{FlowId: id, User: user, Vars: vars, Runners: make([]*Runner, 0), WorkspaceFns: fns}
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
	FlowId int
	WorkspaceFns []*WorkspaceMacro
}

type Runner struct {
	Cancelled bool
}

