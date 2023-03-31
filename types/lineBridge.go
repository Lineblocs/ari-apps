package types

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

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
		if item != nil {
			item.Channel.Hangup()
		}
	}
	b.Bridge.Delete()
}

func (b *LineBridge) AddChannel(channel *LineChannel) {
	b.Channels = append(b.Channels, channel)
}

func (b *LineBridge) RemoveChannel(channel *LineChannel) {
	channels := make([]*LineChannel, 0)
	for _, item := range b.Channels {
		if item.Channel.ID() != channel.Channel.ID() {
			channels = append(channels, item)
		}
	}
	b.Channels = channels
}

func (b *LineBridge) StartWaitingForRingTimeout(timeout int, wg *sync.WaitGroup, ringTimeoutChan <-chan bool) {
	fmt.Println("starting ring timeout checker..")
	fmt.Println("timeout set for: " + strconv.Itoa(timeout))
	duration := time.Now().Add(time.Duration(timeout) * time.Second)

	// Create a context that is both manually cancellable and will signal
	// a cancel at the specified duration.
	ringCtx, cancel := context.WithDeadline(context.Background(), duration)
	defer cancel()
	wg.Done()
	for {
		select {
		case <-ringTimeoutChan:
			fmt.Println("bridge in session. stopping ring timeout")
			return
		case <-ringCtx.Done():
			fmt.Println("Ring timeout elapsed.. ending all calls")
			b.EndBridgeCall()
			return
		}
	}
}
