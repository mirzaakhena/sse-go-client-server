package middleware

import (
	"context"
	"shared/core"
)

func Retry[R any, S any](actionHandler core.ActionHandler[R, S], attempt int) core.ActionHandler[R, S] {
	return func(ctx context.Context, request R) (*S, error) {

		count := 1

		for {
			response, err := actionHandler(ctx, request)
			if err != nil {

				count++
				if count <= attempt {
					continue
				} else {
					return nil, err
				}

			}

			return response, nil
		}

	}
}
