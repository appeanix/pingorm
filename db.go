package pingorm

import (
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var db *gorm.DB

func openDb() (*gorm.DB, error) {
	if db == nil {
		var err error

		//open connection
		if db, err = gorm.Open(mysql.Open(dbConString), &gorm.Config{
			NamingStrategy: schema.NamingStrategy{SingularTable: true},
			Logger:         logger.Default,
		}); err != nil {
			return nil, err
		}

		//setup connection concurrency
		rawDB, err := db.DB()
		if err != nil {
			return nil, err
		}

		rawDB.SetConnMaxLifetime(1 * time.Hour)
		rawDB.SetMaxIdleConns(200)
		rawDB.SetMaxOpenConns(300)

	} else {
		rawDB, err := db.DB()
		if err != nil {
			db = nil
			return nil, err
		}
		if err := rawDB.Ping(); err != nil {
			db = nil
			return nil, err
		}
	}

	return db, nil
}
