package types

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
	SourceLinks *[]Link
	TargetLinks *[]Link
}

type Model struct {
	Id string
}
type Graph struct {
	Cells []*GraphCell `json:"cells"`
}
type FlowVars struct {
	Graph Graph `json:"graph"`
}

type FlowDIDData struct {
	//FlowJson FlowVars `json:"flow_json"`
	WorkspaceId int `json:"workspace_id"`
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
	var model *Model
	sourceLinks := make( []Link, 0 )
	targetLinks := make( []Link, 0 )
	for _, item := range flow.Models {
		if (item.Id == cell.Cell.Id) {
			model = item
		}
	}	

	cell.Model = model

	for _, item := range flow.Vars.Graph.Cells {
		if item.Type == "devs.FlowLink" {
			if item.Source.Id == cell.Cell.Id {
				destCell := findCellInFlow( item.Target.Id, flow, channel )
				link := Link{
					Link: item,
					Source: cell,
					Target: destCell }
				sourceLinks = append( sourceLinks, link )
			} else if item.Target.Id == cell.Cell.Id {
				srcCell := findCellInFlow( item.Target.Id, flow, channel )
				link := Link{
					Link: item,
					Source: srcCell,
					Target: cell }
				targetLinks = append( sourceLinks, link )
			}

		}
	}
	cell.SourceLinks = &sourceLinks
	cell.TargetLinks = &targetLinks
}
func addCellToFlow(id string, flow *Flow, channel *LineChannel) {

	for _, cell := range flow.Cells {
		if cell.Cell.Id == id {
			return
		}
	}

	cellInFlow := findCellInFlow(id, flow, channel)
	flow.Cells = append(flow.Cells, cellInFlow)
}
func NewFlow(user *User, vars *FlowVars, channel *LineChannel) (*Flow) {
	flow := &Flow{User: user, Vars: vars}
	// create cells from flow.Vars
	for _, cell := range flow.Vars.Graph.Cells {
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
	Vars *FlowVars
}

