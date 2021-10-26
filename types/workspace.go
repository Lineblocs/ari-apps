package types
type WorkspaceMacro struct {
	Title string `json:"title"`	
	Function string `json:"function"`	
	CompiledCode string `json:"compiled_code"`	
}
type Workspace struct {
	Id int
	Name string
	Domain string
}