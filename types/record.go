package types

type Record struct {
	Bridge *LineBridge
	Channel *LineChannel
	User *User
	Call *Call
	IsBridge bool
}

func NewRecording(user *User, channel *LineChannel, isBridge bool) (*Record) {
	record := Record{
		Channel: channel,
		User: user,
		IsBridge: isBridge }

	return &record
}