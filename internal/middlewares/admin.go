package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AdminMiddleware(adminSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		secret := c.GetHeader("X-Admin-Secret")
		if secret == "" || secret != adminSecret {
			c.JSON(http.StatusUnauthorized, gin.H{"errors": []string{"unauthorized"}})
			c.Abort()
			return
		}
		c.Next()
	}
}
