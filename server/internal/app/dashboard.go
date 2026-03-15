package app

import (
	"context"
	"net/http"
	"time"

	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/gin-gonic/gin"
)

const dashboardStreamInterval = 1 * time.Second

type dashboardSnapshot struct {
	Metrics     services.HostMetrics    `json:"metrics"`
	Projects    []projectResponse       `json:"projects"`
	Approvals   []model.ApprovalRequest `json:"approvals"`
	Websites    []model.Website         `json:"websites"`
	GeneratedAt time.Time               `json:"generated_at"`
}

func (a *App) handleHostSummary(c *gin.Context) {
	summary, err := a.host.Summary(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (a *App) handleHostMetrics(c *gin.Context) {
	metrics, err := a.host.Metrics(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (a *App) handleDashboardStream(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")

	pushSnapshot := func() {
		snapshot, err := a.dashboardSnapshot(c.Request.Context())
		if err != nil {
			writeSSE(c, "warning", gin.H{"error": err.Error()})
			return
		}
		writeSSE(c, "snapshot", snapshot)
	}

	pushSnapshot()

	ticker := time.NewTicker(dashboardStreamInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			pushSnapshot()
		}
	}
}

func (a *App) dashboardSnapshot(ctx context.Context) (dashboardSnapshot, error) {
	metrics, err := a.host.Metrics(ctx)
	if err != nil {
		return dashboardSnapshot{}, err
	}

	projects, err := a.listProjectResponses(ctx)
	if err != nil {
		return dashboardSnapshot{}, err
	}

	approvals, err := a.listApprovals()
	if err != nil {
		return dashboardSnapshot{}, err
	}

	websites, err := a.listWebsites()
	if err != nil {
		return dashboardSnapshot{}, err
	}

	return dashboardSnapshot{
		Metrics:     metrics,
		Projects:    projects,
		Approvals:   approvals,
		Websites:    websites,
		GeneratedAt: time.Now().UTC(),
	}, nil
}

func (a *App) GetHostSummary(ctx context.Context) (services.HostSummary, error) {
	return a.host.Summary(ctx)
}
