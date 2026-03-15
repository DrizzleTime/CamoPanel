package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"camopanel/server/internal/model"
	"camopanel/server/internal/services"

	"github.com/google/uuid"
)

func (a *App) createApprovalFromProposal(actorID string, proposed *services.ProposedAction) (model.ApprovalRequest, error) {
	if proposed == nil {
		return model.ApprovalRequest{}, fmt.Errorf("缺少 proposed_action")
	}

	switch proposed.Action {
	case model.ApprovalActionDeploy:
		return a.createDeployApproval(actorID, "ai", createProjectRequest{
			Name:       proposed.ProjectName,
			TemplateID: proposed.TemplateID,
			Parameters: proposed.Parameters,
		})
	case model.ApprovalActionStart, model.ApprovalActionStop, model.ApprovalActionRestart, model.ApprovalActionDelete, model.ApprovalActionRedeploy:
		project, err := a.findProject(proposed.ProjectID)
		if err != nil {
			return model.ApprovalRequest{}, fmt.Errorf("AI 提议的项目不存在")
		}
		return a.createProjectActionApproval(actorID, "ai", project, proposed.Action)
	default:
		return model.ApprovalRequest{}, fmt.Errorf("AI 提议了不支持的动作: %s", proposed.Action)
	}
}

func (a *App) saveApproval(actorID, source, action, targetType, targetID string, payload any, summary string) (model.ApprovalRequest, error) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return model.ApprovalRequest{}, err
	}

	approval := model.ApprovalRequest{
		ID:          uuid.NewString(),
		Source:      source,
		Action:      action,
		TargetType:  targetType,
		TargetID:    targetID,
		PayloadJSON: string(rawPayload),
		Summary:     summary,
		Status:      model.ApprovalStatusPending,
		CreatedBy:   actorID,
	}
	if err := a.db.Create(&approval).Error; err != nil {
		return model.ApprovalRequest{}, err
	}
	_ = a.recordAudit(actorID, "approval_created", targetType, targetID, map[string]any{
		"approval_id": approval.ID,
		"action":      action,
		"source":      source,
	})
	return approval, nil
}

func (a *App) approveRequest(ctx context.Context, approvalID, actorID string) (model.ApprovalRequest, error) {
	var approval model.ApprovalRequest
	if err := a.db.First(&approval, "id = ?", approvalID).Error; err != nil {
		return model.ApprovalRequest{}, fmt.Errorf("审批单不存在")
	}
	if approval.Status != model.ApprovalStatusPending {
		return model.ApprovalRequest{}, fmt.Errorf("审批单当前状态不可批准")
	}

	now := time.Now()
	approval.Status = model.ApprovalStatusExecuting
	approval.ApprovedBy = actorID
	approval.ExecutedAt = &now
	if err := a.db.Save(&approval).Error; err != nil {
		return model.ApprovalRequest{}, err
	}

	err := a.executeApproval(ctx, approval)
	if err != nil {
		approval.Status = model.ApprovalStatusFailed
		approval.ErrorMessage = err.Error()
		_ = a.recordAudit(actorID, "approval_failed", approval.TargetType, approval.TargetID, map[string]any{
			"approval_id": approval.ID,
			"error":       err.Error(),
		})
	} else {
		approval.Status = model.ApprovalStatusApproved
		approval.ErrorMessage = ""
		_ = a.recordAudit(actorID, "approval_approved", approval.TargetType, approval.TargetID, map[string]any{
			"approval_id": approval.ID,
			"action":      approval.Action,
		})
	}

	if saveErr := a.db.Save(&approval).Error; saveErr != nil {
		return model.ApprovalRequest{}, saveErr
	}
	if err != nil {
		return approval, err
	}
	return approval, nil
}

func (a *App) rejectRequest(approvalID, actorID, reason string) (model.ApprovalRequest, error) {
	var approval model.ApprovalRequest
	if err := a.db.First(&approval, "id = ?", approvalID).Error; err != nil {
		return model.ApprovalRequest{}, fmt.Errorf("审批单不存在")
	}
	if approval.Status != model.ApprovalStatusPending {
		return model.ApprovalRequest{}, fmt.Errorf("审批单当前状态不可拒绝")
	}

	now := time.Now()
	approval.Status = model.ApprovalStatusRejected
	approval.ApprovedBy = actorID
	approval.ExecutedAt = &now
	approval.ErrorMessage = strings.TrimSpace(reason)
	if err := a.db.Save(&approval).Error; err != nil {
		return model.ApprovalRequest{}, err
	}
	_ = a.recordAudit(actorID, "approval_rejected", approval.TargetType, approval.TargetID, map[string]any{
		"approval_id": approval.ID,
		"reason":      reason,
	})
	return approval, nil
}

func (a *App) executeApproval(ctx context.Context, approval model.ApprovalRequest) error {
	switch approval.Action {
	case model.ApprovalActionDeploy:
		var payload deployApprovalPayload
		if err := json.Unmarshal([]byte(approval.PayloadJSON), &payload); err != nil {
			return err
		}
		return a.executeDeploy(ctx, payload)
	case model.ApprovalActionCreateWebsite:
		var payload createWebsiteApprovalPayload
		if err := json.Unmarshal([]byte(approval.PayloadJSON), &payload); err != nil {
			return err
		}
		return a.executeCreateWebsite(ctx, payload)
	case model.ApprovalActionStart, model.ApprovalActionStop, model.ApprovalActionRestart, model.ApprovalActionDelete, model.ApprovalActionRedeploy:
		var payload projectActionPayload
		if err := json.Unmarshal([]byte(approval.PayloadJSON), &payload); err != nil {
			return err
		}
		return a.executeProjectAction(ctx, payload)
	default:
		return fmt.Errorf("未知审批动作: %s", approval.Action)
	}
}

func (a *App) listApprovals() ([]model.ApprovalRequest, error) {
	var approvals []model.ApprovalRequest
	if err := a.db.Order("created_at desc").Find(&approvals).Error; err != nil {
		return nil, err
	}
	return approvals, nil
}
