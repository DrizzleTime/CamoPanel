package app

import (
	"camopanel/server/internal/model"

	"gorm.io/gorm"
)

func cleanupLegacyApprovalData(db *gorm.DB) error {
	if err := db.Where("action LIKE ?", "approval_%").Delete(&model.AuditEvent{}).Error; err != nil {
		return err
	}

	if !db.Migrator().HasTable("approval_requests") {
		return nil
	}

	return db.Migrator().DropTable("approval_requests")
}
