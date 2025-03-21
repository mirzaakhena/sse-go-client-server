package gateway

import (
	"context"

	"shared/core"
	"shared/utility"
)

type SendSSEMessageReq struct {
	EventType string
	Data      any
}

type SendSSEMessageRes struct{}

type SendSSEMessage = core.ActionHandler[SendSSEMessageReq, SendSSEMessageRes]

func ImplSendSSEMessage(sse *utility.SSEServer) SendSSEMessage {
	return func(ctx context.Context, request SendSSEMessageReq) (*SendSSEMessageRes, error) {

		if sse == nil {
			return &SendSSEMessageRes{}, nil
		}

		err := sse.SendToClients(ctx, utility.Message{
			EventType: request.EventType,
			Data:      request.Data,
		})

		if err != nil {
			return nil, err
		}

		return &SendSSEMessageRes{}, nil
	}
}
