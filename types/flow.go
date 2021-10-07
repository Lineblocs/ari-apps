package types

type Vertice struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
type CellConnection struct {
	Id string `json:"id"`
	Port string `json:"port"`
}
type Cell struct {
	Type string `json:"type"`
	Source CellConnection `json:"source"`
	Target CellConnection `json:"target"`
	Vertices []Vertice `json:"vertices"`
}
type Graph struct {
	Cells []Cell `json:"cells"`
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

func NewFlow(user *User, vars *FlowVars) (*Flow) {
	flow := &Flow{User: user, Vars: vars}
	return flow
}


type Flow struct {
	User *User
	Exten string
	CallerId string
	Channel *LineChannel
	RootCall *Call
	Cells []*Cell
	Vars *FlowVars
}

