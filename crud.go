package pingorm

import (
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repo struct {
	Model interface{}
}

func (repo Repo) Create(_db interface{}, ptrToModel interface{}, option QueryOption) (ptrToResult interface{}, err error) {
	db := _db.(*gorm.DB)
	err = db.Create(ptrToModel).Error
	return ptrToModel, err
}

func (repo Repo) Update(_db interface{}, ptrToModel interface{}, option QueryOption) (ptrToResult interface{}, err error) {
	db := _db.(*gorm.DB)

	db = db.Select(option.GetSelectedFields()).Omit(option.GetOmittedFields()...)
	err = db.Updates(ptrToModel).Error
	return ptrToModel, err
}

func (repo Repo) Upsert(_db interface{}, slice interface{}, options QueryOption) (sliceOfResult interface{}, err error) {
	db := _db.(*gorm.DB)
	err = db.Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns(options.GetSelectedFields()),
	}).Omit(options.GetOmittedFields()...).Create(slice).Error

	return slice, err
}

func (repo Repo) Delete(_db interface{}, sliceOfIDs interface{}, option QueryOption) error {

	if reflect.TypeOf(sliceOfIDs).Kind() == reflect.Slice {
		if reflect.ValueOf(sliceOfIDs).Len() == 0 {
			return nil
		}
	} else {
		panic("slice required")
	}

	db := _db.(*gorm.DB)
	if option.HardDelete() {
		db = db.Unscoped()
	}

	return db.Where("id IN ?", sliceOfIDs).Delete(repo.Model).Error
}
