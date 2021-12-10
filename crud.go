package pingorm

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repo struct {
	Model interface{}
}

func (repo Repo) Create(_db interface{}, model interface{}, option QuerySelector) (ptrToModel interface{}, err error) {

	if ptrToModel, err = parseModelToPtr(model); err != nil {
		return nil, err
	}

	db := _db.(*gorm.DB).
		Session(&gorm.Session{NewDB: true}).
		Set("value:update_on_conflict", option.GetUpdatesOnConflict())

	err = db.Select(option.GetSelectedFields()).
		Omit(option.GetOmittedFields()...).
		Create(ptrToModel).Error

	return ptrToModel, err
}

func (repo Repo) Update(_db interface{}, model interface{}, option QuerySelector) (ptrToModel interface{}, err error) {

	if ptrToModel, err = parseModelToPtr(model); err != nil {
		return nil, err
	}

	db := _db.(*gorm.DB).
		Session(&gorm.Session{NewDB: true}).
		Set("value:update_on_conflict", option.GetUpdatesOnConflict())

	db = db.Select(option.GetSelectedFields()).Omit(option.GetOmittedFields()...)
	err = db.Updates(ptrToModel).Error
	return ptrToModel, err
}

func (repo Repo) Upsert(_db interface{}, slice interface{}, options QuerySelector) (sliceOfResult interface{}, err error) {
	db := _db.(*gorm.DB)
	err = db.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns(options.GetSelectedFields()),
	}).Omit(options.GetOmittedFields()...).Create(slice).Error

	return slice, err
}

func (repo Repo) Delete(_db interface{}, sliceOfIDs interface{}, option QuerySelector) error {

	if reflect.TypeOf(sliceOfIDs).Kind() == reflect.Slice {
		if reflect.ValueOf(sliceOfIDs).Len() == 0 {
			return nil
		}
	} else {
		panic("slice required")
	}

	db := _db.(*gorm.DB)
	if option.IsHardDelete() {
		db = db.Unscoped()
	}

	return db.Where("id IN ?", sliceOfIDs).Delete(repo.Model).Error
}

func (repo Repo) Updates(_db interface{}, sliceOfIDs interface{}, values interface{}, option QuerySelector) error {

	if reflect.TypeOf(sliceOfIDs).Kind() == reflect.Slice {
		if reflect.ValueOf(sliceOfIDs).Len() == 0 {
			return nil
		}
	} else {
		panic("slice required")
	}

	db := _db.(*gorm.DB)
	db = db.Select(option.GetSelectedFields()).Omit(append(option.GetOmittedFields(), clause.Associations)...)

	return db.Where("id IN ?", sliceOfIDs).Updates(values).Error
}

func (repo Repo) Get(_db interface{}, sliceOfIDs interface{}, option QuerySelector) (sliceT interface{}, err error) {
	var ptrSliceT interface{}
	db := _db.(*gorm.DB)
	for _, v := range option.GetPreloadedFields() {
		db = db.Preload(v)
	}

	whereExpr, whereArgs, err := buildWhereExprByKeys(db, sliceOfIDs, option)
	if err != nil {
		return nil, err
	}
	var queryExpr interface{}
	if queryKeys := option.GetKeys(); len(queryKeys) == 0 || len(queryKeys) == 1 {
		queryExpr = db.Where(whereExpr, whereArgs)
	} else {
		queryExpr = db.Where(whereExpr, whereArgs...)
	}

	if mt := reflect.TypeOf(repo.Model); mt.Kind() == reflect.Ptr {
		ptrSliceT = reflect.New(
			reflect.MakeSlice(reflect.SliceOf(mt.Elem()), 0, 0).Type(),
		).Interface()

	} else {
		ptrSliceT = reflect.New(
			reflect.MakeSlice(reflect.SliceOf(mt), 0, 0).Type(),
		).Interface()
	}

	db.Model(repo.Model).
		Select(option.GetSelectedFields()).
		Omit(option.GetOmittedFields()...).Where(queryExpr).
		Find(ptrSliceT)

	sliceT = reflect.ValueOf(ptrSliceT).Elem().Interface()
	return sliceT, err
}

func buildWhereExprByKeys(db *gorm.DB, sliceOfKeyVals interface{}, option QuerySelector) (string, []interface{}, error) {
	var keyCols []string
	for _, key := range option.GetKeys() {
		keyCols = append(keyCols, db.NamingStrategy.ColumnName("", key))
	}
	keyLength := len(keyCols)

	// Reflect the value of first dimension slice in order to iterate through slice
	sliceValues := reflect.ValueOf(sliceOfKeyVals)

	// Build condition expression of empty option keys
	if keyLength == 0 {
		if err := assertSingleDimenSlice(sliceOfKeyVals); err != nil {
			return "", nil, err
		}

		// Table is expected to have unique key column id if keys are not specify.
		var argVals []interface{}
		whereExpr := fmt.Sprintf("id IN ?")
		for i := 0; i < sliceValues.Len(); i++ {
			args := sliceValues.Index(i).Interface()
			argVals = append(argVals, args)
		}

		return whereExpr, argVals, nil
	}

	if err := assert2DimenSlice(sliceOfKeyVals); err != nil {
		return "", nil, err
	}

	// Build condition expression of option keys with only one field
	// i.e: QueryOption{Keys: []string{"keyA"}}
	if keyLength == 1 {
		var argVals []interface{}
		for valIdx := 0; valIdx < sliceValues.Len(); valIdx++ {
			sliceVals := sliceValues.Index(valIdx)

			// assert the length of each slice2DValues match the key's length
			if sliceVals.Len() != keyLength {
				return "", nil, fmt.Errorf("key length %v requires value length %v", keyLength, keyLength)
			}

			argVal := sliceVals.Index(0).Interface()
			argVals = append(argVals, argVal)
		}
		whereExpr := fmt.Sprintf("%s IN ?", keyCols[0])
		return whereExpr, argVals, nil
	}

	var whereCond []string
	var argVals []interface{}
	for valIdx := 0; valIdx < sliceValues.Len(); valIdx++ {
		slice2DValues := sliceValues.Index(valIdx)

		// assert the length of each slice2DValues match the key's length
		if slice2DValues.Len() != keyLength {
			return "", nil, fmt.Errorf("key length %v requires value length %v", keyLength, keyLength)
		}

		// Build individual conditional expression with AND operator i.e (col1Key = ? AND col2Key = ?)
		var condExpr []string
		for keyIndex := 0; keyIndex < keyLength; keyIndex++ {
			condExpr = append(condExpr, fmt.Sprintf("%s = ?", keyCols[keyIndex]))

			argValue := slice2DValues.Index(keyIndex).Interface()
			argVals = append(argVals, argValue)
		}

		fieldExps := strings.Join(condExpr, " AND ")
		whereCond = append(whereCond, fmt.Sprintf("(%s)", fieldExps))
	}

	// Finally, build where expression i.e (col1Key = ? AND col2Key = ?) OR (col1Key = ? AND col2Key = ?)
	whereExpr := strings.Join(whereCond, " OR ")
	return whereExpr, argVals, nil
}

func assertSingleDimenSlice(sliceOfValues interface{}) error {
	argType := reflect.TypeOf(sliceOfValues)
	if err := assertSliceType(argType); err != nil {
		return err
	}

	if argType.Elem().Kind() == reflect.Slice {
		return errors.New("value must be a single dimension slice")
	}

	return nil
}

func assert2DimenSlice(sliceOfSlice interface{}) error {
	argType := reflect.TypeOf(sliceOfSlice)
	if err := assertSliceType(argType); err != nil {
		return err
	}

	sliceElemType := argType.Elem()
	if sliceElemType.Kind() != reflect.Slice {
		return errors.New("value must be 2 dimension slice")
	}

	if slice2DElemType := sliceElemType.Elem(); slice2DElemType.Kind() == reflect.Slice {
		return errors.New("value must be 2 dimension slice")
	}

	return nil
}

func assertSliceType(valueType reflect.Type) error {
	if valueType.Kind() != reflect.Slice {
		return errors.New("value must be a kind of slice")
	}

	return nil
}

func parseModelToPtr(model interface{}) (interface{}, error) {
	if modelType := reflect.TypeOf(model); modelType.Kind() == reflect.Struct {
		ptrToModelVal := reflect.New(modelType)
		ptrToModelVal.Elem().Set(reflect.ValueOf(model))
		return ptrToModelVal.Interface(), nil

	} else if modelType.Kind() == reflect.Ptr && modelType.Elem().Kind() == reflect.Struct {
		return model, nil

	}
	return nil, errors.New("model must be a kind of struct or pointer to struct type")
}
