package types

import (
	"github.com/CyCoreSystems/ari/v5"
)
type LineBridge struct {
	Bridge *ari.BridgeHandle
	AutomateLegAHangup bool
	AutomateLegBHangup bool
	Channels []*LineChannel
	ChannelsToAdd []*LineChannel
}

func NewBridge( bridge *ari.BridgeHandle ) *LineBridge {
	value := LineBridge{Bridge: bridge, Channels: make( []*LineChannel, 0 )}
	return &value
}