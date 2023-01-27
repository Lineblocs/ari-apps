package types

import (
	"errors"
	"fmt"
	"strconv"
	"testing"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/client/native"
	"github.com/stretchr/testify/require"
)

func NewMockClient() ari.Client {
	var cl ari.Client
	host := "155.138.140.32"
	ariUrl := fmt.Sprintf("http://%s:8088/ari", host)
	wsUrl := fmt.Sprintf("ws://%s:8088/ari/events", host)
	cl, _ = native.Connect(&native.Options{
		Application:  "lineblocs",
		Username:     "ariuser",
		Password:     "*9lwK@992I1gjK1P",
		URL:          ariUrl,
		WebsocketURL: wsUrl})
	return cl
}

func NewMockChannel(cl ari.Client) *ari.ChannelHandle {
	channelRequest := ari.ChannelCreateRequest{
		Endpoint: "SIP/" + "80011972598400495" + "@159.89.124.168",
		App:      "lineblocs",
		AppArgs:  "DID_DIAL,"}
	outboundChannel, _ := cl.Channel().Create(nil, channelRequest)
	return outboundChannel

}

var channelHandler = NewMockChannel(NewMockClient())

func TestSafeHangup(t *testing.T) {
	t.Parallel()
	type fields struct {
		lineChannel LineChannel
	}
	type want struct {
		err error
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "OK",
			fields: fields{
				lineChannel: LineChannel{
					Channel: channelHandler,
				},
			},
			want: want{
				err: nil,
			},
		},
		{
			name: "NilChannel",
			fields: fields{
				lineChannel: LineChannel{
					Channel: nil,
				},
			},
			want: want{
				err: errors.New("No Channel is existed."),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fields.lineChannel.SafeHangup()
			require.Equal(t, err, tt.want.err)
		})
	}
}

func TestAnswer(t *testing.T) {
	t.Parallel()
	type fields struct {
		lineChannel LineChannel
	}
	type want struct {
		err error
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "OK",
			fields: fields{
				lineChannel: LineChannel{
					Channel: channelHandler,
				},
			},
			want: want{
				err: nil,
			},
		},
		{
			name: "NilChannel",
			fields: fields{
				lineChannel: LineChannel{
					Channel: nil,
				},
			},
			want: want{
				err: errors.New("No Channel is existed."),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.fields.lineChannel.Answer()
			require.Equal(t, err, tt.want.err)
		})
	}
}

func TestCreateCall(t *testing.T) {
	params := CallParams{
		From:        "123123234",
		To:          "80011972598400495",
		Status:      "start",
		Direction:   "outbound",
		UserId:      2,
		WorkspaceId: 3,
		ChannelId:   "test-channel"}

	t.Parallel()
	type fields struct {
		lineChannel LineChannel
		id          string
		params      CallParams
	}
	type want struct {
		err  error
		call *Call
	}
	tests := []struct {
		name   string
		fields fields
		want   want
	}{
		{
			name: "OK",
			fields: fields{
				lineChannel: LineChannel{
					Channel: channelHandler,
				},
				id:     "1",
				params: params,
			},
			want: want{
				err: nil,
			},
		},
		{
			name: "NoIntCallId",
			fields: fields{
				id:     "1a",
				params: params,
			},
			want: want{
				call: nil,
				err: &strconv.NumError{
					Func: "Atoi",
					Num:  "1a",
					Err:  strconv.ErrSyntax,
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fields.lineChannel.CreateCall(tt.fields.id, &tt.fields.params)
			require.Equal(t, err, tt.want.err)
		})
	}
}

func TestStartWaitingForRingTimeout(t *testing.T) {

}
