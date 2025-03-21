package gateway

import (
	"context"
	"server/model"
	"server/utility"
	"shared/core"

	"gorm.io/gorm"
)

type ClientSaveReq struct {
	Client model.Client
}

type ClientSaveRes struct{}

type ClientSave = core.ActionHandler[ClientSaveReq, ClientSaveRes]

func ImplClientSaveWithSQlite(db *gorm.DB) ClientSave {
	return func(ctx context.Context, req ClientSaveReq) (*ClientSaveRes, error) {

		if err := utility.GetDBFromContext(ctx, db).Save(req.Client).Error; err != nil {
			return nil, err
		}

		return &ClientSaveRes{}, nil
	}
}
