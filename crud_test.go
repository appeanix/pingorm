package pingorm

import (
	"errors"

	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/khaiql/dbcleaner"
	"github.com/khaiql/dbcleaner/engine"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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
		seeds            []interface{}
		input            interface{}
		inputQueryParams QueryOption
		expGot           interface{}
		expDbAuthor      []Author
		expDbBook        []Book
		expDbEditor      []Editor
		expErr           interface{}
	}{
		// It should create only Author
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

		// It should create Author along with a new associated Book
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

		// It should
		// 1. create Author,
		// 2. update existing Book (ID 1),
		// 3. update association Editor (ID 1),
		// 4. create a new associated Book
		// 5. create a new associated Editor
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Vichheka",
					Sex:  "Male",
				},
				&Editor{
					ID:   1,
					Name: "HengLong",
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
				Name: "New Author",
				Sex:  "Male",
				Books: []Book{
					{
						ID:    1,
						Title: "Hello-World-Updated",
						Editor: Editor{
							ID:   1,
							Name: "HengLong-Updated",
						},
					},
					{
						Title: "New Book",
						Editor: Editor{
							Name: "New Editor",
							Sex:  "Female",
						},
					},
				},
			},
			inputQueryParams: QueryOption{
				UpdatesOnConflict: map[string][]string{
					"Book": {
						"AuthorID",
						"Title",
					},
					"Editor": {
						"Name",
					},
				},
			},
			expGot: &Author{
				ID:   2,
				Name: "New Author",
				Sex:  "Male",
				Books: []Book{
					{
						ID:       1,
						Title:    "Hello-World-Updated",
						AuthorID: 2,
						EditorID: 1,
						Editor: Editor{
							ID:   1,
							Name: "HengLong-Updated",
						},
					},
					{
						ID:       2,
						Title:    "New Book",
						AuthorID: 2,
						EditorID: 2,
						Editor: Editor{
							ID:   2,
							Name: "New Editor",
							Sex:  "Female",
						},
					},
				},
			},
			expDbAuthor: []Author{
				{
					ID:   1,
					Name: "Vichheka",
					Sex:  "Male",
				},
				{
					ID:   2,
					Name: "New Author",
					Sex:  "Male",
				},
			},
			expDbBook: []Book{
				{
					ID:       1,
					AuthorID: 2,
					EditorID: 1,
					Title:    "Hello-World-Updated",
				},
				{
					ID:       2,
					AuthorID: 2,
					EditorID: 2,
					Title:    "New Book",
				},
			},
			expDbEditor: []Editor{
				{
					ID:   1,
					Name: "HengLong-Updated",
					Sex:  "Male",
				},
				{
					ID:   2,
					Name: "New Editor",
					Sex:  "Female",
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
			inputQueryParams: QueryOption{SelectedFields: []string{"ID", "Name", "Sex", "ContactNumber"}},
			expErr:           nil,
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
			inputQueryParams: QueryOption{OmittedFields: []string{"Books"}},
			expErr:           nil,
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

	for _, tc := range tests[2:3] {
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

			got, err := Repo{}.Create(db, tc.input, tc.inputQueryParams)

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
				ID:   1,
				Name: "Henglong-Updated",
				Sex:  "Male-Updated",
				Books: []Book{
					{
						Title:    "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expGot: &Author{
				ID:   1,
				Name: "Henglong-Updated",
				Sex:  "Male-Updated",
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
					Name: "Henglong-Updated",
					Sex:  "Male-Updated",
				},
			},
			expDbBook: []Book{
				{
					ID:       1,
					Title:    "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			expDbEditor: []Editor{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			queryParams: QueryOption{SelectedFields: []string{"ID", "Name", "Sex"}},
		},

		// It should update all fields of Author except an omitted field and create its new associated Book when OmittedF is specified
		{
			seeds: []interface{}{
				&Author{
					ID:            1,
					Name:          "Henglong",
					Sex:           "Male",
					ContactNumber: "1234567890",
				},
				&Editor{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Book{
					ID:       1,
					Title:    "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			input: Author{
				ID:            1,
				Name:          "Henglong-Updated",
				Sex:           "Male-Updated",
				ContactNumber: "1234567890-Updated",
				Books: []Book{
					{
						Title:    "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expGot: &Author{
				ID:            1,
				Name:          "Henglong-Updated",
				Sex:           "Male-Updated",
				ContactNumber: "1234567890-Updated",
				Books: []Book{
					{
						ID:       2,
						Title:    "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expDbAuthor: []Author{
				{
					ID:            1,
					Name:          "Henglong-Updated",
					Sex:           "Male-Updated",
					ContactNumber: "1234567890",
				},
			},
			expDbBook: []Book{
				{
					ID:       1,
					Title:    "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
				{
					ID:       2,
					Title:    "New-Book",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			expDbEditor: []Editor{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			queryParams: QueryOption{OmittedFields: []string{"ContactNumber"}},
		},

		// It should update fields of Author and only create its new associated Book when SelectedF and OmittedF are not specified
		{
			seeds: []interface{}{
				&Author{
					ID:            1,
					Name:          "Henglong",
					Sex:           "Male",
					ContactNumber: "1234567890",
				},
				&Editor{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Book{
					ID:       1,
					Title:    "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			input: Author{
				ID:            1,
				Name:          "Henglong-Updated",
				Sex:           "Male-Updated",
				ContactNumber: "1234567890-Updated",
				Books: []Book{
					{
						ID:       1,
						Title:    "Hello-World-Updated",
						AuthorID: 1,
						EditorID: 1,
					},
					{
						Title:    "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expGot: &Author{
				ID:            1,
				Name:          "Henglong-Updated",
				Sex:           "Male-Updated",
				ContactNumber: "1234567890-Updated",
				Books: []Book{
					{
						ID:       1,
						Title:    "Hello-World-Updated",
						AuthorID: 1,
						EditorID: 1,
					},
					{
						ID:       2,
						Title:    "New-Book",
						AuthorID: 1,
						EditorID: 1,
					},
				},
			},
			expDbAuthor: []Author{
				{
					ID:            1,
					Name:          "Henglong-Updated",
					Sex:           "Male-Updated",
					ContactNumber: "1234567890-Updated",
				},
			},
			expDbBook: []Book{
				{
					ID:       1,
					Title:    "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
				{
					ID:       2,
					Title:    "New-Book",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			expDbEditor: []Editor{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			queryParams: QueryOption{},
		},

		// It should
		// 1. Update Author,
		// 2. update existing Book (ID 1),
		// 3. update association Editor (ID 1),
		// 4. create a new associated Book
		// 5. create a new associated Editor
		{
			seeds: []interface{}{
				&Author{
					ID:            1,
					Name:          "Henglong",
					Sex:           "Male",
					ContactNumber: "1234567890",
				},
				&Editor{
					ID:   1,
					Name: "Lego",
					Sex:  "Male",
				},
				&Book{
					ID:       1,
					Title:    "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			input: Author{
				ID:            1,
				Name:          "Henglong-Updated",
				Sex:           "Male-Updated",
				ContactNumber: "1234567890-Updated",
				Books: []Book{
					{
						ID:       1,
						Title:    "Hello-World-Updated",
						AuthorID: 1,
						EditorID: 1,
						Editor: Editor{
							ID:   1,
							Name: "Lego-Updated",
						},
					},
					{
						Title: "New-Book",
						Editor: Editor{
							Name: "New Editor",
							Sex:  "Female",
						},
					},
				},
			},
			expGot: &Author{
				ID:            1,
				Name:          "Henglong-Updated",
				Sex:           "Male-Updated",
				ContactNumber: "1234567890-Updated",
				Books: []Book{
					{
						ID:       1,
						Title:    "Hello-World-Updated",
						AuthorID: 1,
						EditorID: 1,
						Editor: Editor{
							ID:   1,
							Name: "Lego-Updated",
						},
					},
					{
						ID:       2,
						Title:    "New-Book",
						AuthorID: 1,
						EditorID: 2,
						Editor: Editor{
							ID:   2,
							Name: "New Editor",
							Sex:  "Female",
						},
					},
				},
			},
			expDbAuthor: []Author{
				{
					ID:            1,
					Name:          "Henglong-Updated",
					Sex:           "Male-Updated",
					ContactNumber: "1234567890-Updated",
				},
			},
			expDbBook: []Book{
				{
					ID:       1,
					Title:    "Hello-World-Updated",
					AuthorID: 1,
					EditorID: 1,
				},
				{
					ID:       2,
					Title:    "New-Book",
					AuthorID: 1,
					EditorID: 2,
				},
			},
			expDbEditor: []Editor{
				{
					ID:   1,
					Name: "Lego-Updated",
					Sex:  "Male",
				},
				{
					ID:   2,
					Name: "New Editor",
					Sex:  "Female",
				},
			},
			queryParams: QueryOption{
				UpdatesOnConflict: map[string][]string{
					"Book": {
						"AuthorID",
						"Title",
					},
					"Editor": {
						"Name",
					},
				},
			},
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

func TestDelete(t *testing.T) {
	mockTime := time.Now()

	tests := []struct {
		seeds        []interface{}
		input        interface{}
		deletedModel interface{}
		expGot       interface{}
		expDbAuthor  []Author
		queryParams  QueryOption
		expErr       error
	}{
		//soft delete
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
				&Author{
					ID:   3,
					Name: "NaNa",
					Sex:  "Female",
				},
			},
			input: []uint32{
				1,
				2,
			},
			expGot: []uint32{
				1,
				2,
			},
			deletedModel: &Author{},
			expDbAuthor: []Author{
				{

					ID:      1,
					Name:    "Henglong",
					Sex:     "Male",
					Deleted: gorm.DeletedAt{Time: mockTime.Round(time.Millisecond), Valid: true},
				},
				{
					ID:      2,
					Name:    "Vicheka",
					Sex:     "Male",
					Deleted: gorm.DeletedAt{Time: mockTime.Round(time.Millisecond), Valid: true},
				},
				{
					ID:   3,
					Name: "NaNa",
					Sex:  "Female",
				},
			},
		},

		//hard delete
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
				&Author{
					ID:   3,
					Name: "NaNa",
					Sex:  "Female",
				},
			},
			input: []uint32{
				1,
				2,
			},
			expGot: []uint32{
				1,
				2,
			},
			deletedModel: &Author{},
			expDbAuthor: []Author{
				{
					ID:   3,
					Name: "NaNa",
					Sex:  "Female",
				},
			},
			queryParams: QueryOption{HardDelete: true},
		},

		// Test the input is empty
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
				&Author{
					ID:   3,
					Name: "NaNa",
					Sex:  "Female",
				},
			},
			input:        []uint32{},
			expGot:       []uint32{},
			deletedModel: &Author{},
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
				{
					ID:   3,
					Name: "NaNa",
					Sex:  "Female",
				},
			},
			expErr: nil,
		},
	}

	for _, tc := range tests {
		func() {
			req := require.New(t)

			cleanTables()

			db, err := openDb()
			req.Nil(err)
			db = db.Debug()

			db.Config.NowFunc = func() time.Time {
				return mockTime
			}

			for _, seed := range tc.seeds {
				err = db.Create(seed).Error
				req.Nil(err)
			}

			errDelete := Repo{Model: tc.deletedModel}.Delete(db, tc.input, tc.queryParams)

			req.Nil(errDelete)
			req.Equal(tc.expErr, errDelete)

			var dbAuthors []Author
			db.Model(&Author{}).Unscoped().Select("Deleted", "ID", "Name", "Sex").Find(&dbAuthors)
			req.Equal(tc.expDbAuthor, dbAuthors)

		}()
	}
}

func TestUpdates(t *testing.T) {

	tests := []struct {
		seeds       []interface{}
		inputIDs    interface{}
		expGot      interface{}
		expDbAuthor []Author
		expDbBook   []Book
		expDbEditor []Editor
		inputValues interface{}
		queryParams QueryOption
	}{
		//Updates Name and Sex where ID = 1
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
			},
			inputIDs: []uint32{
				1,
			},
			expGot: []uint32{
				1,
			},
			inputValues: &Author{
				Name: "Henglong-Updated",
				Sex:  "Male-Updated",
			},
			expDbAuthor: []Author{
				{
					ID:   1,
					Name: "Henglong-Updated",
					Sex:  "Male-Updated",
				},
				{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
			},
			expDbBook:   []Book{},
			expDbEditor: []Editor{},
		},

		// Updates Sex where ID in (1,2)
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
			},
			inputIDs: []uint32{
				1,
				2,
			},
			expGot: []uint32{
				1,
				2,
			},
			inputValues: &Author{
				Sex: "Male-Updated",
			},
			expDbAuthor: []Author{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male-Updated",
				},
				{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male-Updated",
				},
			},
			expDbBook:   []Book{},
			expDbEditor: []Editor{},
		},

		// It should updates fields of Author specified in the SeletedF
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
			},
			inputIDs: []uint32{
				1,
				2,
			},
			expGot: []uint32{
				1,
				2,
			},
			inputValues: &Author{
				Name: "Henglong-Updated",
				Sex:  "Male-Updated",
			},
			expDbAuthor: []Author{
				{
					ID:   1,
					Name: "Henglong-Updated",
					Sex:  "Male",
				},
				{
					ID:   2,
					Name: "Henglong-Updated",
					Sex:  "Male",
				},
			},
			expDbBook:   []Book{},
			expDbEditor: []Editor{},
			queryParams: QueryOption{SelectedFields: []string{"Name"}},
		},

		// It should not create associated records.
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
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
			inputIDs: []uint32{
				1,
			},
			expGot: []uint32{
				1,
			},
			inputValues: &Author{
				Name: "Henglong-Updated",
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
					Name: "Henglong-Updated",
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

			errUpdates := Repo{}.Updates(db, tc.inputIDs, tc.inputValues, tc.queryParams)

			req.Nil(errUpdates)

			var dbAuthors []Author
			db.Model(&Author{}).Select("ID", "Name", "Sex").Find(&dbAuthors)
			req.Equal(tc.expDbAuthor, dbAuthors)

			var dbBooks []Book
			db.Model(&Book{}).Select("ID", "Title", "AuthorID", "EditorID").Find(&dbBooks)
			req.Equal(tc.expDbBook, dbBooks)

			var dbEditors []Editor
			db.Model(&Editor{}).Select("ID", "Name", "Sex").Find(&dbEditors)
			req.Equal(tc.expDbEditor, dbEditors)

		}()
	}
}

func TestGet(t *testing.T) {

	tests := []struct {
		seeds       []interface{}
		inputIDs    interface{}
		expGot      interface{}
		queryParams QueryOption
		model       interface{}
	}{
		//Get Author where ID = 1
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
			},
			inputIDs: []uint32{
				1,
			},
			expGot: []Author{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
			},
			model: &Author{},
		},

		//Get Author where ID in (1,2)
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
			},
			inputIDs: []uint32{
				1,
				2,
			},
			expGot: []Author{
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
			model: &Author{},
		},

		// It should Get fields of Author specified in the SeletedF
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
			},
			inputIDs: []uint32{
				1,
				2,
			},
			expGot: []Author{
				{
					ID:   1,
					Name: "Henglong",
				},
				{
					ID:   2,
					Name: "Vicheka",
				},
			},
			model:       &Author{},
			queryParams: QueryOption{SelectedFields: []string{"Name", "ID"}},
		},

		// It should Get all fields of Author except an omitted field
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
			},
			inputIDs: []uint32{
				1,
				2,
			},
			expGot: []Author{
				{
					ID:  1,
					Sex: "Male",
				},
				{
					ID:  2,
					Sex: "Male",
				},
			},
			model:       &Author{},
			queryParams: QueryOption{OmittedFields: []string{"Name"}},
		},

		//It should Get fields of Author and it association speciied in the PreloadedF
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
				&Editor{
					ID:   1,
					Name: "Vicheka",
					Sex:  "Male",
				},
				&Book{
					ID:       1,
					Title:    "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			inputIDs: []uint32{
				1,
			},
			expGot: []Author{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
					Books: []Book{
						{
							ID: 1,
							AuthorID: 1,
							EditorID: 1,
							Title: "Hello-World",
						},
					},
				},
			},
			queryParams: QueryOption{PreloadedFields: []string{"Books"}},
			model: &Author{},
		},

		//It should Get fields of Author and it association speciied in the PreloadedF
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
				&Editor{
					ID:   1,
					Name: "Vicheka",
					Sex:  "Male",
				},
				&Book{
					ID:       1,
					Title:    "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			inputIDs: []uint32{
				1,
			},
			expGot: []Author{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
					Books: []Book{
						{
							ID: 1,
							AuthorID: 1,
							EditorID: 1,
							Title: "Hello-World",
							Editor: Editor{
								ID: 1,
								Name: "Vicheka",
								Sex: "Male",
							},
						},
					},
				},
			},
			queryParams: QueryOption{PreloadedFields: []string{"Books.Editor"}},
			model: &Author{},
		},

		//It should Get fields of Author and it association speciied in the PreloadedF
		{
			seeds: []interface{}{
				&Author{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
				},
				&Author{
					ID:   2,
					Name: "Vicheka",
					Sex:  "Male",
				},
				&Editor{
					ID:   1,
					Name: "Vicheka",
					Sex:  "Male",
				},
				&Book{
					ID:       1,
					Title:    "Hello-World",
					AuthorID: 1,
					EditorID: 1,
				},
			},
			inputIDs: []uint32{
				1,
			},
			expGot: []Author{
				{
					ID:   1,
					Name: "Henglong",
					Sex:  "Male",
					Books: []Book{
						{
							ID: 1,
							Title: "Hello-World",
							EditorID: 1,
							AuthorID: 1,
						},
					},
				},
			},
			queryParams: QueryOption{PreloadedFields: []string{"Books.Title"}},
			model: &Author{},
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

			got, errGet := Repo{Model: tc.model}.Get(db, tc.inputIDs, tc.queryParams)

			req.Nil(errGet)
			req.Equal(tc.expGot, got)

			var dbAuthors []Author
			db.Model(&Author{}).Select("ID", "Name", "Sex").Find(&dbAuthors)

		}()
	}
}
