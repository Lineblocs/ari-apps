package types

import (
	"time"
)
type StatusParams struct {
	Status string `json:"status"`
	CallId int `json:"call_id"`
	Ip string `json:"ip"`
}
type CallParams struct {
	From string `json:"from"`
	To string `json:"to"`
	Status string `json:"status"`
	Direction string `json:"direction"`
	UserId int `json:"user_id"`
	WorkspaceId int `json:"workspace_id"`
}
type Call struct {
	Bridge *LineBridge
	UserId int
	CallId int
	Channel *LineChannel
	Started time.Time
	Ended time.Time
	Params *CallParams
}
func (c *Call) GetStartTime() (int) {
	return 0
}
func (c *Call)  FigureOutEndedTime() (int) {
	return 0
}