package bootstrap

import (
	"github.com/gin-gonic/gin"
)

func DefaultMiddlewares() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		gin.Logger(),
		gin.Recovery(),
	}
}
