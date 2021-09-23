package pingorm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {

	req := require.New(t)
	db, err := openDb()
	req.Nil(err)

	// Migrate models here
	db.AutoMigrate(&Author{})
	db.AutoMigrate(&Editor{})
	db.AutoMigrate(&Book{})
}
