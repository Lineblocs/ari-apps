package types

import (
	"testing"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/rid"
	"github.com/stretchr/testify/require"
)

func TestAddChannel(t *testing.T) {
	var bridge *ari.BridgeHandle
	var err error
	lineChannel := LineChannel{
		Channel: NewMockChannel(NewMockClient()),
	}
	src := lineChannel.Channel.Key()
	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
	bridge, err = NewMockClient().Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		bridge = nil
	}
	lineBridge := NewBridge(bridge)

	t.Parallel()
	type fields struct {
		lineBridge  *LineBridge
		lineChannel *LineChannel
	}
	type want struct {
		length int
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "OK",
			fields: fields{
				lineBridge:  lineBridge,
				lineChannel: &lineChannel,
			},
			want: want{
				1,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.lineBridge.AddChannel(tt.fields.lineChannel)
			require.Equal(t, len(tt.fields.lineBridge.Channels), tt.want.length)
		})
	}
}

func TestRemoveChannel(t *testing.T) {
	var bridge *ari.BridgeHandle
	var err error
	lineChannel := LineChannel{
		Channel: NewMockChannel(NewMockClient()),
	}
	src := lineChannel.Channel.Key()
	key := src.New(ari.BridgeKey, rid.New(rid.Bridge))
	bridge, err = NewMockClient().Bridge().Create(key, "mixing", key.ID)
	if err != nil {
		bridge = nil
	}
	lineBridge := NewBridge(bridge)
	outChannel := LineChannel{
		Channel: NewMockChannel(NewMockClient()),
	}
	lineBridge.AddChannel(&outChannel)

	t.Parallel()
	type fields struct {
		lineBridge  *LineBridge
		lineChannel *LineChannel
	}
	type want struct {
		length int
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "OK",
			fields: fields{
				lineBridge:  lineBridge,
				lineChannel: &lineChannel,
			},
			want: want{
				1,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.lineBridge.RemoveChannel(tt.fields.lineChannel)
			require.Equal(t, len(tt.fields.lineBridge.Channels), tt.want.length)
		})
	}
}
