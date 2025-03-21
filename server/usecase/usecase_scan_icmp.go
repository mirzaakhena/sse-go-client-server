package usecase

import (
	"context"
	"server/gateway"
	"shared/core"
)

type ScanICMPTriggerReq struct {
	ClientIDs []string `json:"client_ids"`
}

type ScanICMPTriggerRes struct{}

// Send to All Client
type ScanICMPTrigger = core.ActionHandler[ScanICMPTriggerReq, ScanICMPTriggerRes]

func ImplScanICMPTrigger(
	SendSSEMessage gateway.SendSSEMessage,
) ScanICMPTrigger {
	return func(ctx context.Context, req ScanICMPTriggerReq) (*ScanICMPTriggerRes, error) {

		// send and forget
		_, err := SendSSEMessage(ctx, gateway.SendSSEMessageReq{
			EventType: "scan_icmp",
			// Data:      req.IPRange,
		})

		if err != nil {
			return nil, err
		}

		return &ScanICMPTriggerRes{}, nil
	}
}
