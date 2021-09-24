package pingorm

import (
	"errors"
	"testing"

	"github.com/go-sql-driver/mysql"
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

func TestCreate(t *testing.T) {

	tests := []struct {
		seeds       []interface{}
		input       interface{}
		expGot      interface{}
		expDbAuthor []Author
		expDbBook   []Book
		expDbEditor []Editor
		queryParams QueryOption
		expErr      interface{}
	}{
		//Create only the Author
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			input: Author{
				Name: "Vicheka",
				Sex: "Male",
			},
			expGot: &Author{
				ID: 2,
				Name: "Vicheka",
				Sex: "Male",
			},
			expDbAuthor: []Author{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
			},
			expDbBook: []Book{},
			expDbEditor: []Editor{},
			queryParams: QueryOption{},
			expErr: nil,
		},

		//create one more associated objects
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Editor{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Book{
					ID: 1,
					AuthorID: 1,
					EditorID: 1,
					Title: "Hello-World",
				},
			},
			input: Book{
				Title: "New-Book",
				AuthorID: 1,
				EditorID: 1,
			},
			expGot: &Book{
				ID:   2,
				Title: "New-Book",
				AuthorID: 1,
				EditorID: 1,
			},
			expDbAuthor: []Author{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			expDbBook: []Book{
				{
					ID:       1,
					AuthorID: 1,
					EditorID: 1,
					Title:    "Hello-World",
				},
				{
					ID:       2,
					AuthorID: 1,
					EditorID: 1,
					Title:    "New-Book",
				},
			},
			expDbEditor: []Editor{
				{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
				},
			},
			queryParams: QueryOption{SelectedFields: []string{}},
			expErr: nil,
		},

		//Gorm ignore updating the existing associated objects
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Editor{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Book{
					ID: 1,
					AuthorID: 1,
					EditorID: 1,
					Title: "Hello-World",
				},
			},
			input: Author{
				Name: "Vicheka",
				Sex: "Male",
				Books: []Book{
					{
						ID: 1,
						Title: "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expGot: &Author{
				ID: 2,
				Name: "Vicheka",
				Sex: "Male",
				Books: []Book{
					{
						ID: 1,
						Title: "New-Book",
						AuthorID: 2,
						EditorID: 1,
					},
				},
			},
			expDbAuthor: []Author{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
			},
			expDbBook: []Book{
				{
					ID:       1,
					AuthorID: 2,
					EditorID: 1,
					Title:    "Hello-World",
				},
			},
			expDbEditor: []Editor{
				{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
				},
			},
			queryParams: QueryOption{SelectedFields: []string{}},
			expErr: nil,
		},

		//skip creating all associations
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Editor{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Book{
					ID: 1,
					AuthorID: 1,
					EditorID: 1,
					Title: "Hello-World",
				},
			},
			input: Author{
				Name: "Vicheka",
				Sex: "Male",
				Books: []Book{
					{
						Title: "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expGot: &Author{
				ID: 2,
				Name: "Vicheka",
				Sex: "Male",
				Books: []Book{
					{
						Title: "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expDbAuthor: []Author{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
			},
			expDbBook: []Book{
				{
					ID:       1,
					AuthorID: 1,
					EditorID: 1,
					Title:    "Hello-World",
				},
			},
			expDbEditor: []Editor{
				{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
				},
			},
			queryParams: QueryOption{SelectedFields: []string{"id","name","sex"}},
			expErr: nil,
		},

		// returns error when MySql inserts duplicate target record.
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			input: Author{
				ID:   1,
					Name: "Henglong",
					Sex:  "Male",
			},
			expGot: &Author{
				ID:   1,
				Name: "Henglong",
				Sex:  "Male",
			},
			expDbAuthor: []Author{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			expDbBook: []Book{},
			expDbEditor: []Editor{},
			expErr: &mysql.MySQLError{Number: 0x0426 ,Message: "Duplicate entry '1' for key 'author.PRIMARY'"},
		},
	}

	for _, tc := range tests {
		func() {
			req := require.New(t)

			cleanTables()

			db, err := openDb()
			req.Nil(err)
			db = db.Debug()

			for _, seed := range tc.seeds {
				err = db.Create(seed).Error
				req.Nil(err)
			}

			got, err := Repo{}.Create(db, tc.input, tc.queryParams)

			req.Equal(tc.expGot, got)
			req.Equal(tc.expErr, err)

			var dbAuthors []Author
			db.Model(&Author{}).Select("id", "name", "sex").Find(&dbAuthors)
			req.Equal(tc.expDbAuthor, dbAuthors)

			var dbBooks []Book
			db.Model(&Book{}).Select("id", "title", "author_id", "editor_id").Find(&dbBooks)
			req.Equal(tc.expDbBook, dbBooks)

			var dbEditors []Editor
			db.Model(&Editor{}).Select("id", "name", "sex").Find(&dbEditors)
			req.Equal(tc.expDbEditor, dbEditors)

		}()
	}
}
