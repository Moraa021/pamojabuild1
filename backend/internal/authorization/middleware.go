package authorization

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"pamojabuild1/backend/internal/apihttp"
)

type TrusteeRelationshipReader interface {
	IsTaskTrustee(ctx context.Context, taskSlug string, userID int64) (bool, error)
}

// RequireTaskTrustee protects trustee-only operations such as reviewing or
// signing a payout. Administrators do not bypass this check: possession of a
// global admin capability must never count as a trustee's financial approval.
func RequireTaskTrustee(reader TrusteeRelationshipReader) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetInt64("user_id")
		taskSlug := c.Param("slug")
		allowed, err := reader.IsTaskTrustee(c.Request.Context(), taskSlug, userID)
		if err != nil {
			apihttp.WriteError(c, http.StatusInternalServerError, apihttp.CodeInternal, "authorization check failed")
			return
		}
		if !allowed {
			apihttp.WriteError(c, http.StatusForbidden, apihttp.CodeUnauthorized, "task trustee access required")
			return
		}
		c.Next()
	}
}
