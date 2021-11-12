package types

import (
	//"github.com/CyCoreSystems/ari/v5"
)
type LineConference struct {
	Bridge *LineBridge
	WaitingParticipants []*LineChannel
	Participants []*LineChannel
	ModeratorInConf bool
	Id string
	User* User
}

func NewConference( id string, user *User, bridge *LineBridge ) *LineConference {
	value := LineConference{Id: id, User: user, Bridge: bridge}
	return &value
}