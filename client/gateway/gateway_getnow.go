package gateway

import (
	"context"
	"shared/core"
	"time"
)

type GetNowReq struct{}

type GetNowRes struct {
	Now time.Time
}

type GetNow = core.ActionHandler[GetNowReq, GetNowRes]

func ImplGetNow() GetNow {
	return func(ctx context.Context, request GetNowReq) (*GetNowRes, error) {
		return &GetNowRes{Now: time.Now()}, nil
	}
}
