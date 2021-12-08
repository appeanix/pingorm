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
	if mt := reflect.TypeOf(repo.Model); mt.Kind() == reflect.Ptr {
		ptrSliceT = reflect.New(
			reflect.MakeSlice(reflect.SliceOf(mt.Elem()), 0, 0).Type(),
		).Interface()

	} else {
		ptrSliceT = reflect.New(
			reflect.MakeSlice(reflect.SliceOf(mt), 0, 0).Type(),
		).Interface()
	}

	db := _db.(*gorm.DB)

	for _, v := range option.GetPreloadedFields() {
		db = db.Preload(v)
	}
	db.Model(repo.Model).
		Select(option.GetSelectedFields()).
		Omit(option.GetOmittedFields()...).Where("id IN ?", sliceOfIDs).
		Find(ptrSliceT)

	sliceT = reflect.ValueOf(ptrSliceT).Elem().Interface()
	return sliceT, err
}

func buildWhereExprByKeys(sliceOfKeyVals interface{}, option QueryOption) (string, []interface{}, error) {
	keyCols := option.GetKeys()
	keyLength := len(keyCols)
	var whereExpression string
	var whereArgs []interface{}

	if keyLength == 0 {
		if err := assertSingleDimenSlice(sliceOfKeyVals); err != nil {
			return "", nil, err
		}

		whereExpression = fmt.Sprintf("id IN ?")
		whereArgs = append(whereArgs, sliceOfKeyVals)

		return whereExpression, whereArgs, nil
	}

	if err := assert2DimenSlice(sliceOfKeyVals); err != nil {
		return "", nil, err
	}

	// assert the query keys to lowercase
	keyCols, err := parseSliceValueToLowerCase(keyCols)
	if err != nil {
		return "", nil, err
	}

	// Retrieve the value of 2DimenSlice to build the conditional expression
	sliceValues := reflect.ValueOf(sliceOfKeyVals)
	var whereExpr []string

	for i := 0; i < sliceValues.Len(); i++ {
		slice2DValues := sliceValues.Index(i)

		if slice2DValues.Len() != keyLength {
			return "", nil, fmt.Errorf("key length %v requires value length %v", keyLength, keyLength)
		}

		if keyLength == 1 {
			whereExpression = fmt.Sprintf("%s IN ?", keyCols[0])
			return whereExpression, []interface{}{slice2DValues.Interface()}, nil
		}

		// Build individual conditional expression with AND operator i.e (col1Key = ? AND col2Key = ?)
		var condExpr []string
		for keyIndex := 0; keyIndex < keyLength; keyIndex++ {
			condExpr = append(condExpr, fmt.Sprintf("%s = ?", keyCols[keyIndex]))

			argValue := slice2DValues.Index(keyIndex).Interface()
			whereArgs = append(whereArgs, argValue)
		}

		fieldExps := strings.Join(condExpr, " AND ")
		whereExpr = append(whereExpr, fmt.Sprintf("(%s)", fieldExps))
	}

	// Finally, build where expression i.e (col1Key = ? AND col2Key = ?) OR (col1Key = ? AND col2Key = ?)
	whereExpression = strings.Join(whereExpr, " OR ")

	return whereExpression, whereArgs, nil
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

func parseSliceValueToLowerCase(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, errors.New("values must include one or more elements")
	}

	var valueToLower []string
	for i := range values {
		strToLower := strings.ToLower(values[i])
		valueToLower = append(valueToLower, strToLower)
	}
	return valueToLower, nil
}
