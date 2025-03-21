package gateway

import (
	"context"
	"server/model"
	"server/utility"
	"shared/core"

	"gorm.io/gorm"
)

type ClientGetOneReq struct {
	ClientID string
}

type ClientGetOneRes struct {
	Client model.Client
}

type ClientGetOne = core.ActionHandler[ClientGetOneReq, ClientGetOneRes]

func ImplClientGetOneWithSQlite(db *gorm.DB) ClientGetOne {
	return func(ctx context.Context, req ClientGetOneReq) (*ClientGetOneRes, error) {

		var client model.Client

		if err := utility.GetDBFromContext(ctx, db).First(&client, req.ClientID).Error; err != nil {
			return nil, err
		}

		return &ClientGetOneRes{Client: client}, nil
	}
}
