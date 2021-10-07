package types

type Record struct {
	Bridge *LineBridge
	Channel *LineChannel
	User *User
	Call *Call
	IsBridge bool
}