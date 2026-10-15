package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"camopanel/backend/internal/host"
	"github.com/gin-gonic/gin"
)

func mountHost(api *gin.RouterGroup, sampler host.Sampler) {
	api.GET("/host/overview", func(c *gin.Context) {
		overview, err := sampler.Overview(time.Now())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, overview)
	})
	api.GET("/host/processes", func(c *gin.Context) {
		limit := 10
		if raw := c.Query("limit"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > 50 {
				badRequest(c, "进程数量须在 1–50 之间")
				return
			}
			limit = n
		}
		sortBy := c.DefaultQuery("sort", "cpu")
		if sortBy != "cpu" && sortBy != "mem" {
			badRequest(c, "排序只支持 cpu 或 mem")
			return
		}
		list, err := sampler.Processes(time.Now(), limit, sortBy)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"processes": list})
	})
}
