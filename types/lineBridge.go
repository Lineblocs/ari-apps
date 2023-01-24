package types

import (
	"github.com/CyCoreSystems/ari/v5"
)

type LineBridge struct {
	Bridge             *ari.BridgeHandle
	AutomateLegAHangup bool
	AutomateLegBHangup bool
	Channels           []*LineChannel
	ChannelsToAdd      []*LineChannel
}

func NewBridge(bridge *ari.BridgeHandle) *LineBridge {
	value := LineBridge{Bridge: bridge, Channels: make([]*LineChannel, 0)}
	return &value
}

func (b *LineBridge) EndBridgeCall() {
	for _, item := range b.Channels {
		//utils.Log(logrus.DebugLevel,"ending call: " + item.Channel.Key().ID)
		if item != nil {
			item.Channel.Hangup()
		}
	}
	b.Bridge.Delete()
}
