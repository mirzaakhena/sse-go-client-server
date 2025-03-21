package middleware

import (
	"context"
	"server/utility"
	"shared/core"

	"gorm.io/gorm"
)

func TransactionMiddleware[R any, S any](actionHandler core.ActionHandler[R, S], db *gorm.DB) core.ActionHandler[R, S] {
	return func(ctx context.Context, request R) (*S, error) {
		var result *S
		var err error

		txErr := db.Transaction(func(tx *gorm.DB) error {

			// Create a new context with the transaction
			txCtx := context.WithValue(ctx, utility.GormDBKey, tx)

			// Call the action handler within the transaction
			result, err = actionHandler(txCtx, request)
			if err != nil {
				// If there's an error, return it to roll back the transaction
				return err
			}

			// If everything is okay, return nil to commit the transaction
			return nil
		})

		if txErr != nil {
			// If there was an error in the transaction, return it
			return nil, txErr
		}

		// Return the result and error from the action handler

		return result, err
	}
}
