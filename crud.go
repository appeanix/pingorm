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

	whereExpr, whereArgs, err := buildWhereExprByKeys(db, sliceOfIDs, option)
	if err != nil {
		return err
	}

	return db.Where(whereExpr, whereArgs).Delete(repo.Model).Error
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
		Omit(option.GetOmittedFields()...).Where(whereExpr, whereArgs).
		Find(ptrSliceT)

	sliceT = reflect.ValueOf(ptrSliceT).Elem().Interface()
	return sliceT, err
}

func buildWhereExprByKeys(db *gorm.DB, sliceOfKeyVals interface{}, option QuerySelector) (string, interface{}, error) {
	var keyCols []string
	for _, key := range option.GetKeys() {
		keyCols = append(keyCols, db.NamingStrategy.ColumnName("", key))
	}
	keyLength := len(keyCols)

	// Reflect the value of first dimension slice in order to iterate through slice
	sliceValues := reflect.ValueOf(sliceOfKeyVals)

	// Build condition expression of empty or only one field option keys
	if keyLength <= 1 {
		if err := assertSingleDimenSlice(sliceOfKeyVals); err != nil {
			return "", nil, err
		}

		var key string
		if keyLength == 0 {
			// Table is expected to have unique key column id if keys are not specify.
			key = "id"
		} else {
			// Build condition expression of option keys with only one field
			// i.e: QueryOption{Keys: []string{"keyA"}}
			key = keyCols[0]
		}

		var argVals []interface{}
		for sliceIdx := 0; sliceIdx < sliceValues.Len(); sliceIdx++ {
			arg := sliceValues.Index(sliceIdx).Interface()
			argVals = append(argVals, arg)
		}

		whereExpr := fmt.Sprintf("%s IN ?", key)
		return whereExpr, argVals, nil
	}

	if err := assert2DimenSlice(sliceOfKeyVals); err != nil {
		return "", nil, err
	}

	// Build argument values of IN multiple columns
	// i.e: "(colA, ColB) IN ?"
	keyExpr := strings.Join(keyCols, ", ")
	whereExpr := fmt.Sprintf("(%s) IN ?", keyExpr)

	var argVals [][]interface{}
	for valIdx := 0; valIdx < sliceValues.Len(); valIdx++ {
		slice2DVal := sliceValues.Index(valIdx)

		// assert the length of each slice2DValues match the key's length
		if slice2DVal.Len() != keyLength {
			return "", nil, fmt.Errorf("key length %v requires value length %v", keyLength, keyLength)
		}

		var slices []interface{}
		for keyIndex := 0; keyIndex < keyLength; keyIndex++ {
			slice := slice2DVal.Index(keyIndex).Interface()
			slices = append(slices, slice)

		}
		argVals = append(argVals, slices)
	}
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
