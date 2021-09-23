package pingorm

import (
	"errors"
	"testing"

	"github.com/khaiql/dbcleaner"
	"github.com/khaiql/dbcleaner/engine"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestParseModelToPtr(t *testing.T) {
	tests := []struct {
		model  interface{}
		expGot interface{}
		expErr error
	}{
		{
			model: Author{
				Name: "Vichheka",
			},
			expGot: &Author{
				Name: "Vichheka",
			},
			expErr: nil,
		},
		{
			model: &Author{
				Name: "Vichheka",
			},
			expGot: &Author{
				Name: "Vichheka",
			},
			expErr: nil,
		},
		{
			model:  "hello",
			expGot: nil,
			expErr: errors.New("model must be a kind of struct or pointer to struct type"),
		},
	}

	for _, tc := range tests {
		req := require.New(t)

		got, err := parseModelToPtr(tc.model)

		req.Equal(tc.expErr, err)
		req.Equal(tc.expGot, got)
	}
}

func TestCleanTables(t *testing.T) {
	tests := []struct {
		seeds []interface{}
	}{
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Mr. A",
				},
				&Editor{
					ID:   1,
					Name: "Mr. B",
				},
				&Book{
					EditorID: 1,
					AuthorID: 1,
					Title:    "Pingorm",
				},
			},
		},
	}

	for _, tc := range tests {
		req := require.New(t)

		db, err := openDb()
		req.Nil(err)

		for _, seed := range tc.seeds {
			err = db.Clauses(clause.OnConflict{
				UpdateAll: true,
			}).Create(seed).Error
			req.Nil(err)
		}

		cleanTables()

		const zero int64 = 0
		var authorCount int64
		db.Model(&Author{}).Count(&authorCount)
		req.Equal(zero, authorCount)

		var editorCount int64
		db.Model(&Editor{}).Count(&editorCount)
		req.Equal(zero, editorCount)

		var bookCount int64
		db.Model(&Book{}).Count(&bookCount)
		req.Equal(zero, bookCount)
	}
}

func cleanTables() {
	mysql := engine.NewMySQLEngine(dbConString)
	cleaner := dbcleaner.New()
	cleaner.SetEngine(mysql)
	cleaner.Clean(modelsToTableNames()...)
	cleaner.Close()
}
