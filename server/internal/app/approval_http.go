package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type rejectApprovalRequest struct {
	Reason string `json:"reason"`
}

func (a *App) handleApprovals(c *gin.Context) {
	approvals, err := a.listApprovals()
	if err != nil {
		writeError(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": approvals})
}

func (a *App) handleApprove(c *gin.Context) {
	approval, err := a.approveRequest(c.Request.Context(), c.Param("id"), currentUser(c).ID)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"approval": approval})
}

func (a *App) handleReject(c *gin.Context) {
	var req rejectApprovalRequest
	_ = c.ShouldBindJSON(&req)

	approval, err := a.rejectRequest(c.Param("id"), currentUser(c).ID, req.Reason)
	if err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"approval": approval})
}
