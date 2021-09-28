package pingorm

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func openDb() (*gorm.DB, error) {

	var err error
	var db *gorm.DB

	//open connection
	if db, err = gorm.Open(mysql.Open(dbConString), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
		Logger:         logger.Default,
	}); err != nil {
		return nil, err
	}

	if err := db.Callback().Create().Before("gorm:create").Register("app:update_on_conflict", func(tx *gorm.DB) {

		if val, isSet := tx.Get("value:update_on_conflict"); isSet {
			if updateOnConflict, ok := val.(map[string][]string); ok {
				schemaName := tx.Statement.Schema.Name

				if fieldsToUpdate, ok := updateOnConflict[schemaName]; ok {
					columnsToUpdate := make([]string, len(fieldsToUpdate))
					for i := range fieldsToUpdate {
						columnsToUpdate[i] = tx.NamingStrategy.ColumnName("", fieldsToUpdate[i])
					}

					tx.Clauses(clause.OnConflict{
						DoUpdates: clause.AssignmentColumns(columnsToUpdate),
					})
				}
			}
		}

	}); err != nil {
		return nil, err
	}

	return db, nil
}
