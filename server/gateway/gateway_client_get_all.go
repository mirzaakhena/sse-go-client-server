package gateway

import (
	"context"
	"server/model"
	"server/utility"
	"shared/core"

	"gorm.io/gorm"
)

type ClientGetAllReq struct {
}

type ClientGetAllRes struct {
	Clients []model.Client
}

type ClientGetAll = core.ActionHandler[ClientGetAllReq, ClientGetAllRes]

func ImplClientGetAllWithSQlite(db *gorm.DB) ClientGetAll {
	return func(ctx context.Context, req ClientGetAllReq) (*ClientGetAllRes, error) {

		var clients []model.Client

		if err := utility.GetDBFromContext(ctx, db).Find(&clients).Error; err != nil {
			return nil, err
		}

		return &ClientGetAllRes{Clients: clients}, nil
	}
}
