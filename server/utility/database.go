package utility

import (
	"context"
	"shared/core"

	"gorm.io/gorm"
)

const GormDBKey core.ContextKey = "GORM_DB"

func GetDBFromContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	dbCtx, ok := ctx.Value(GormDBKey).(*gorm.DB)
	if !ok {
		return db
	}
	return dbCtx
}
