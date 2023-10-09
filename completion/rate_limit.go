package completion

import (
	"context"

	"github.com/polyfire/api/db"
	"github.com/polyfire/api/utils"
)

func CheckRateLimit(ctx context.Context) error {
	rateLimitStatus := ctx.Value(utils.ContextKeyRateLimitStatus)

	if rateLimitStatus == db.RateLimitStatusOk {
		return nil
	}

	if rateLimitStatus == db.RateLimitStatusReached {
		return RateLimitReached
	}

	if rateLimitStatus == db.RateLimitStatusProjectReached {
		return ProjectRateLimitReached
	}

	return UnknownError
}
