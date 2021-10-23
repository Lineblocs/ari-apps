package grpc;

import (
	"fmt"
	"golang.org/x/net/context"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type Server struct {
}

func (*Server) CreateBridge(context.Context, *BridgeRequest) (*BridgeReply, error) {
	fmt.Println("creating bridge!!!");
	reply := BridgeReply{}
	return &reply, nil
}
func (*Server) CreateCall(context.Context, *CallRequest) (*CallReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateCall not implemented")
}
func (*Server) AddChannel(context.Context, *ChannelRequest) (*ChannelReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AddChannel not implemented")
}
func (*Server) PlayRecording(context.Context, *RecordingPlayRequest) (*RecordingPlayReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PlayRecording not implemented")
}
func (*Server) GetChannel(context.Context, *ChannelFetchRequest) (*ChannelFetchReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetChannel not implemented")
}
func (*Server) CreateConference(context.Context, *ConferenceRequest) (*ConferenceReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateConference not implemented")
}
func (*Server) ChannelGetBridge(context.Context, *ChannelGetBridgeRequest) (*ChannelGetBridgeReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelGetBridge not implemented")
}
func (*Server) ChannelRemoveFromBridge(context.Context, *ChannelRemoveBridgeRequest) (*ChannelRemoveBridgeReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelRemoveFromBridge not implemented")
}
func (*Server) ChannelPlayTTS(context.Context, *ChannelTTSRequest) (*ChannelTTSReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelPlayTTS not implemented")
}
func (*Server) ChannelStartAcceptingInput(context.Context, *ChannelInputRequest) (*ChannelInputReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelStartAcceptingInput not implemented")
}
func (*Server) ChannelRemoveDTMFListeners(context.Context, *ChannelRemoveDTMFRequest) (*ChannelRemoveDTMFReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelRemoveDTMFListeners not implemented")
}
func (*Server) ChannelAutomateCallHangup(context.Context, *GenericChannelReq) (*GenericChannelResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelAutomateCallHangup not implemented")
}
func (*Server) ChannelGotoFlowWidget(context.Context, *ChannelFlowWidgetRequest) (*ChannelFlowWidgetReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelGotoFlowWidget not implemented")
}
func (*Server) ChannelStartRinging(context.Context, *GenericChannelReq) (*GenericChannelReq, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelStartRinging not implemented")
}
func (*Server) ChannelStopRinging(context.Context, *GenericChannelReq) (*GenericChannelResp, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ChannelStopRinging not implemented")
}
func (*Server) BridgeAddChannel(context.Context, *BridgeChannelRequest) (*BridgeChannelReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BridgeAddChannel not implemented")
}
func (*Server) BridgeAddChannels(context.Context, *BridgeChannelsRequest) (*BridgeChannelsReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BridgeAddChannels not implemented")
}
func (*Server) BridgePlayTTS(context.Context, *BridgeTTSRequest) (*BridgeTTSReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BridgePlayTTS not implemented")
}
func (*Server) BridgeAutomateLegAHangup(context.Context, *BridgeAutomateLegRequest) (*BridgeAutomateLegReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BridgeAutomateLegAHangup not implemented")
}
func (*Server) BridgeAutomateLegBHangup(context.Context, *BridgeAutomateLegRequest) (*BridgeAutomateLegReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BridgeAutomateLegBHangup not implemented")
}
func (*Server) BridgeAttachEventListener(context.Context, *BridgeEventRequest) (*BridgeEventReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BridgeAttachEventListener not implemented")
}
func (*Server) RecordingDeleteRecording(context.Context, *RecordingDeleteRequest) (*RecordingDeleteReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RecordingDeleteRecording not implemented")
}
func (*Server) RecordingAddRecordingTag(context.Context, *RecordingTagRequest) (*RecordingTagReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RecordingAddRecordingTag not implemented")
}
func (*Server) RecordingDeleteRecordingTag(context.Context, *RecordingTagDeleteRequest) (*RecordingTagDeleteReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RecordingDeleteRecordingTag not implemented")
}
func (*Server) SessionListRecordings(context.Context, *SessionRecordingsRequest) (*SessionRecordingsReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SessionListRecordings not implemented")
}
func (*Server) ConferenceAddWaitingParticipant(context.Context, *ConferenceParticipantRequest) (*ConferenceParticipantReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConferenceAddWaitingParticipant not implemented")
}
func (*Server) ConferenceAddParticipant(context.Context, *ConferenceParticipantRequest) (*ConferenceParticipantReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConferenceAddParticipant not implemented")
}
func (*Server) ConferenceSetModeratorInConf(context.Context, *ConferenceModeratorRequest) (*ConferenceModeratorReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConferenceSetModeratorInConf not implemented")
}
func (*Server) ConferenceAttachEventListener(context.Context, *ConferenceEventRequest) (*ConferenceEventReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ConferenceAttachEventListener not implemented")
}