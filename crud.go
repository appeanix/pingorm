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

func buildCompositeExpression(_db interface{}, sliceOfValues interface{}, option QueryOption) (*gorm.DB, error) {
	db := _db.(*gorm.DB)

	queryKeys := option.GetKeys()
	keyLength := len(queryKeys)
	var queryClauses string
	var fieldArgs []interface{}

	if keyLength == 0 {
		if err := assertSingleDimenSlice(sliceOfValues); err != nil {
			return nil, err
		}

		queryClauses = fmt.Sprintf("id IN ?")
		fieldArgs = append(fieldArgs, sliceOfValues)
	}

	if keyLength >= 1 {
		if err := assert2DimenSlice(sliceOfValues); err != nil {
			return nil, err
		}

		// {{..}, {..}, {..}}
		sliceValues := reflect.ValueOf(sliceOfValues)
		var qryExp []string

		for i := 0; i < sliceValues.Len(); i++ {
			arrValue := sliceValues.Index(i)

			if arrValue.Len() != keyLength {
				return nil, errors.New("number of slice value must be the same as query's key length")
			}

			if keyLength == 1 {
				queryClauses = fmt.Sprintf("%s IN ?", queryKeys[0])
				return db.Where(queryClauses, arrValue.Interface()), nil
			}

			// build expression: fieldA = ?
			// and arg field value
			var condFieldExp []string
			for keyIndex := 0; keyIndex < keyLength; keyIndex++ {
				condFieldExp = append(condFieldExp, fmt.Sprintf("%s = ?", queryKeys[keyIndex]))

				argValue := arrValue.Index(keyIndex).Interface()
				fieldArgs = append(fieldArgs, argValue)
			}

			// build expression: (fieldA = ? AND fieldB = ?)
			fieldExps := strings.Join(condFieldExp, " AND ")
			qryExp = append(qryExp, fmt.Sprintf("(%s)", fieldExps))
		}

		// build expression: (filedA = ? AND fieldB = ?) OR (fieldC = ? AND fieldD = ?)
		queryClauses = strings.Join(qryExp, " OR ")
	}

	return db.Where(queryClauses, fieldArgs...), nil
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
