package pingorm

import (
	"time"

	"gorm.io/gorm"
)

// Model sample entities here
type (
	Author struct {
		ID            uint32 `gorm:"primaryKey"`
		ContactNumber string
		Name          string
		Sex           string
		Dob           *time.Time
		Deleted       gorm.DeletedAt
		Books         []Book
	}
)

type (
	Editor struct {
		ID      uint32 `gorm:"primaryKey"`
		Name    string
		Sex     string
		Dob     *time.Time
		Deleted gorm.DeletedAt
		Books   []Book
	}
)

type (
	Book struct {
		ID          uint32 `gorm:"primaryKey"`
		Title       string
		PublishDate *time.Time
		AuthorID    uint32
		EditorID    uint32
		Author      Author `gorm:"foreignKey:AuthorID;"`
		Editor      Editor `gorm:"foreignKey:EditorID;"`
		Deleted     gorm.DeletedAt
	}
)

type (
	QueryOption struct {
		SelectedFields    []string
		OmittedFields     []string
		PreloadedFields   []string
		UpdatesOnConflict map[string][]string
		HardDelete        bool
	}

	QuerySelector interface {
		GetSelectedFields() []string
		GetOmittedFields() []string
		GetUpdatesOnConflict() map[string][]string
		IsHardDelete() bool
		GetPreloadedFields() []string
	}
)

func (option QueryOption) GetSelectedFields() []string {
	return option.SelectedFields
}

func (option QueryOption) GetOmittedFields() []string {
	return option.OmittedFields
}

func (option QueryOption) GetUpdatesOnConflict() map[string][]string {
	return option.UpdatesOnConflict
}

func (option QueryOption) IsHardDelete() bool {
	return option.HardDelete
}

func (option QueryOption) GetPreloadedFields() []string {
	return option.PreloadedFields
}
