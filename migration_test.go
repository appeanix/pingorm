package pingorm

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm/schema"
)

var models = []interface{}{
	&Author{},
	&Editor{},
	&Book{},
}

func TestMigrate(t *testing.T) {

	req := require.New(t)
	db, err := openDb()
	req.Nil(err)

	// Migrate models here
	db.AutoMigrate(models...)
}

func modelsToTableNames() []string {
	naming := schema.NamingStrategy{SingularTable: true}
	var tables []string
	for _, model := range models {
		tblName := naming.TableName(reflect.TypeOf(model).Elem().Name())
		tables = append(tables, tblName)
	}
	return tables
}
