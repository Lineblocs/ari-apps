package types

import (
	"github.com/CyCoreSystems/ari/v5"
)
type LineBridge struct {
	Bridge *ari.BridgeHandle
	AutomateLegAHangup bool
	AutomateLegBHangup bool
	Channels []LineChannel
	ChannelsToAdd []LineChannel
}