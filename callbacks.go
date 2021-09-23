package pingorm

import (
	"gorm.io/gorm"
)

const AuditOptionOnCreateKey = "app_audit_on_create"

type AuditOption struct {
	Table         string
	AuditedColumn string
	Skip          bool
}

func RegisterAuditCallbacks(db *gorm.DB, uid string, orgID string) {
	if createCallback := db.Callback().Create(); createCallback.Get("app:creator") == nil {
		createCallback.Before("gorm:create").Register("app:creator", func(tx *gorm.DB) {

			var options []AuditOption
			if opt, has := tx.Get(AuditOptionOnCreateKey); has {
				if singleOption, ok := opt.(AuditOption); ok {
					options = append(options, singleOption)
				}
				if multiOptions, ok := opt.([]AuditOption); ok {
					options = multiOptions
				}
			}

			if _, ok := tx.Statement.Schema.FieldsByDBName["created_by"]; ok {
				if canSetAuditValue(tx.Statement.Schema.Table, "created_by", options) {
					tx.Statement.SetColumn("created_by", uid, true)
				}
			}
			if _, ok := tx.Statement.Schema.FieldsByDBName["org_id"]; ok {
				if canSetAuditValue(tx.Statement.Schema.Table, "org_id", options) {
					tx.Statement.SetColumn("org_id", orgID, true)
				}
			}
		})
	}

	if updateCallback := db.Callback().Update(); updateCallback.Get("app:updater") == nil {
		updateCallback.Before("gorm:update").Register("app:updater", func(tx *gorm.DB) {
			if _, ok := tx.Statement.Schema.FieldsByDBName["updated_by"]; ok {
				tx.Statement.SetColumn("updated_by", uid, true)
			}
		})
	}
}

func canSetAuditValue(targetTable, targetColumn string, options []AuditOption) bool {
	for _, option := range options {
		if option.Table == targetTable && option.AuditedColumn == targetColumn && option.Skip {
			return false
		}
	}
	return true
}
