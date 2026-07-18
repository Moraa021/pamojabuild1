package authorization

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "authorization check failed"})
			c.Abort()
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "task trustee access required"})
			c.Abort()
			return
		}
		c.Next()
	}
}
