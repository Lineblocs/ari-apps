package types
type UserInfo struct {
}
type User struct {
	Id int
	Info UserInfo
	Workspace Workspace
	WorkspaceName string
	Domain string
	Plan string
}