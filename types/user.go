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

func NewUser(userId int, workspaceId int, workspaceName string) (*User) {

	domain := workspaceName + ".lineblocs.com"
	user := User{ Id: userId, Workspace: Workspace{
				Id: workspaceId,
				Name: workspaceName,
				Domain: domain }}
	return &user
}