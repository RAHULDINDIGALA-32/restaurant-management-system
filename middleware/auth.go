package middleware

import (
	"net/http"
	"strings"

	helper "restaurant-management-system/helpers"

	"github.com/gin-gonic/gin"
)

func Authentication() gin.HandlerFunc {
	return func(c *gin.Context) {

		//  Get Authorization header
		authHeader := c.GetHeader("Authorization")

		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header is required",
			})
			return
		}

		//  Expect: Bearer <token>
		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header must use Bearer token",
			})
			return
		}

		tokenString := strings.TrimSpace(parts[1])

		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "access token is required",
			})
			return
		}

		//  Validate access token
		claims, err := helper.ValidateAccessToken(tokenString)

		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired access token",
			})
			return
		}

		//  Store authenticated user's information in Gin context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("first_name", claims.FirstName)
		c.Set("last_name", claims.LastName)
		c.Set("token_type", claims.TokenType)
		c.Set("jti", claims.ID)

		//  Continue to protected route
		c.Next()
	}
}
