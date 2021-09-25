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
				Sex:  "Male",
			},
			expGot: &Author{
				ID:   2,
				Name: "Vicheka",
				Sex:  "Male",
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
			expDbBook:   []Book{},
			expDbEditor: []Editor{},
			expErr:      nil,
		},

		// Create Author along with a new associated Book
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
					ID:       1,
					AuthorID: 1,
					EditorID: 1,
					Title:    "Hello-World",
				},
			},
			input: Author{
				Name: "Vicheka",
				Sex:  "Male",
				Books: []Book{
					{
						Title:    "New-Book",
						EditorID: 1,
						AuthorID: 1,
					},
				},
			},
			expGot: &Author{
				ID:   2,
				Name: "Vicheka",
				Sex:  "Male",
				Books: []Book{
					{
						ID:       2,
						Title:    "New-Book",
						EditorID: 1,
						AuthorID: 2,
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
				{
					ID:       2,
					AuthorID: 2,
					EditorID: 1,
					Title:    "New-Book",
				},
			},
			expDbEditor: []Editor{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			expErr: nil,
		},

		// It should create Author and ignore updating existing Book
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
					ID:       1,
					AuthorID: 1,
					EditorID: 1,
					Title:    "Hello-World",
				},
			},
			input: Author{
				Name: "Vicheka",
				Sex:  "Male",
				Books: []Book{
					{
						ID:       1,
						Title:    "Hello-World-Updated",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expGot: &Author{
				ID:   2,
				Name: "Vicheka",
				Sex:  "Male",
				Books: []Book{
					{
						ID:       1,
						Title:    "Hello-World-Updated",
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
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			expErr: nil,
		},

		//Create Author and ignore creating and updating associations using Select
		{
			seeds: []interface{}{
				&Author{
					ID:            1,
					ContactNumber: "12345678",
					Name:          "Henglong",
					Sex:           "Male",
				},
				&Editor{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Book{
					ID:       1,
					AuthorID: 1,
					EditorID: 1,
					Title:    "Hello-World",
				},
			},
			input: Author{
				Name:          "Vicheka",
				Sex:           "Male",
				ContactNumber: "12345678",
				Books: []Book{
					{
						Title:    "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expGot: &Author{
				ID:            2,
				Name:          "Vicheka",
				Sex:           "Male",
				ContactNumber: "12345678",
				Books: []Book{
					{
						Title:    "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expDbAuthor: []Author{
				{
					ID:            1,
					Name:          "Henglong",
					Sex:           "Male",
					ContactNumber: "12345678",
				},
				{
					ID:            2,
					Name:          "Vicheka",
					Sex:           "Male",
					ContactNumber: "12345678",
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
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			queryParams: QueryOption{SelectedFields: []string{"ID", "Name", "Sex", "ContactNumber"}},
			expErr:      nil,
		},

		//Create Author and ignore creating and updating associations using Omit
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
					ID:       1,
					AuthorID: 1,
					EditorID: 1,
					Title:    "Hello-World",
				},
			},
			input: Author{
				Name: "Vicheka",
				Sex:  "Male",
				Books: []Book{
					{
						Title:    "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expGot: &Author{
				ID:   2,
				Name: "Vicheka",
				Sex:  "Male",
				Books: []Book{
					{
						Title:    "New-Book",
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
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			queryParams: QueryOption{OmittedFields: []string{"Books"}},
			expErr:      nil,
		},

		// Create duplicate Author should return error
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
			expDbBook:   []Book{},
			expDbEditor: []Editor{},
			expErr:      &mysql.MySQLError{Number: 0x0426, Message: "Duplicate entry '1' for key 'author.PRIMARY'"},
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
			db.Model(&Author{}).Select("id", "name", "sex", "contact_number").Find(&dbAuthors)
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

func TestUpdate(t *testing.T) {

	tests := []struct {
		seeds       []interface{}
		input       interface{}
		expGot      interface{}
		expDbAuthor []Author
		expDbBook   []Book
		expDbEditor []Editor
		queryParams QueryOption
	}{
		// It should update fields of Author specified in the SelectedF. New Book is not created.
		{
			seeds: []interface{}{
				&Author{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
				},
				&Editor{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
				},
				&Book{
					ID: 1,
					AuthorID: 1,
					EditorID: 1,
					Title: "Hello-World",
				},	
			},
			input: Author{
				ID: 1,
				Name: "Henglong-Updated",
				Sex: "Male-Updated",
				Books: []Book{
					{
						Title: "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expGot: &Author{
				ID: 1,
				Name: "Henglong-Updated",
				Sex: "Male-Updated",
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
					ID: 1,
					Name: "Henglong-Updated",
					Sex: "Male-Updated",
				},
			},
			expDbBook: []Book{
				{
					ID: 1,
					Title: "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			expDbEditor: []Editor{
				{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
				},
			},
			queryParams: QueryOption{SelectedFields: []string{"ID","Name","Sex"}},
		},
	
		// It should update all fields of Author except an omitted field and create its new associated Book when OmittedF is specified
		{
			seeds: []interface{}{
				&Author{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
					ContactNumber: "1234567890",
				},
				&Editor{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
				},
				&Book{
					ID: 1,
					Title: "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			input: Author{
				ID: 1,
				Name: "Henglong-Updated",
				Sex: "Male-Updated",
				ContactNumber: "1234567890-Updated",
				Books: []Book{
					{
						Title: "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expGot: &Author{
				ID: 1,
				Name: "Henglong-Updated",
				Sex: "Male-Updated",
				ContactNumber: "1234567890-Updated",
				Books: []Book{
					{
						ID: 2,
						Title: "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expDbAuthor: []Author{
				{
					ID: 1,
					Name: "Henglong-Updated",
					Sex: "Male-Updated",
					ContactNumber: "1234567890",
				},
			},
			expDbBook: []Book{
				{
					ID: 1,
					Title: "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
				{
					ID: 2,
					Title: "New-Book",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			expDbEditor: []Editor{
				{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
				},
			},
			queryParams: QueryOption{OmittedFields: []string{"ContactNumber"}},
		},
	
		// It should update fields of Author and only create its new associated Book when SelectedF and OmittedF are not specified
		{
			seeds: []interface{}{
				&Author{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
					ContactNumber: "1234567890",
				},
				&Editor{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
				},
				&Book{
					ID: 1,
					Title: "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			input: Author{
				ID: 1,
				Name: "Henglong-Updated",
				Sex: "Male-Updated",
				ContactNumber: "1234567890-Updated",
				Books: []Book{
					{
						ID: 1,
						Title: "Hello-World-Updated",
						AuthorID: 1,
						EditorID: 1,
					},
					{
						Title: "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expGot: &Author{
				ID: 1,
				Name: "Henglong-Updated",
				Sex: "Male-Updated",
				ContactNumber: "1234567890-Updated",
				Books: []Book{
					{
						ID: 1,
						Title: "Hello-World-Updated",
						AuthorID: 1,
						EditorID: 1,
					},
					{
						ID: 2,
						Title: "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expDbAuthor: []Author{
				{
					ID: 1,
					Name: "Henglong-Updated",
					Sex: "Male-Updated",
					ContactNumber: "1234567890-Updated",
				},
			},
			expDbBook: []Book{
				{
					ID: 1,
					Title: "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
				{
					ID: 2,
					Title: "New-Book",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			expDbEditor: []Editor{
				{
					ID: 1,
					Name: "Henglong",
					Sex: "Male",
				},
			},
			queryParams: QueryOption{},
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

			got, err := Repo{}.Update(db, tc.input, tc.queryParams)

			req.Nil(err)

			req.Equal(tc.expGot, got)

			var dbAuthors []Author
			db.Model(&Author{}).Select("id", "name", "sex","contact_number").Find(&dbAuthors)
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
