package types

import (
	"github.com/CyCoreSystems/ari/v5"
)
type LineChannel struct {
	Channel *ari.ChannelHandle
	LineBridge *LineBridge
	currentCellIndex int
	dtmfPressed string
}

func (channel *LineChannel) RemoveFromBridge( ) {

}