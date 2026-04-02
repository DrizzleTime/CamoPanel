package httpx

import "github.com/gin-gonic/gin"

func OK(c *gin.Context, data any) {
	c.JSON(200, data)
}

func Error(c *gin.Context, status int, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": message})
}

func ErrorFrom(c *gin.Context, err error) {
	status, message := MapError(err)
	Error(c, status, message)
}
