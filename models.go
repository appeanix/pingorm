package pingorm

import "time"

// Model sample entities here
type (
	Author struct {
		ID            uint32 `gorm:"primaryKey"`
		ContactNumber string
		Name          string
		Sex           string
		Dob           *time.Time
		Books         []Book
	}
)

type (
	Editor struct {
		ID    uint32 `gorm:"primaryKey"`
		Name  string
		Sex   string
		Dob   *time.Time
		Books []Book
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
	}
)

type (
	QueryOption struct {
		SelectedFields  []string
		OmittedFields   []string
		PreloadedFields []string
		HardDelete      bool
	}

	QuerySelector interface {
		GetSelectedFields() []string
		GetOmittedFields() []string
		GetPreloadedFields() []string
		IsHardDelete() bool
	}
)

func (option QueryOption) GetSelectedFields() []string {
	return option.SelectedFields
}

func (option QueryOption) GetOmittedFields() []string {
	return option.OmittedFields
}

func (option QueryOption) GetPreloadedFields() []string {
	return option.PreloadedFields
}

func (option QueryOption) IsHardDelete() bool {
	return option.HardDelete
}
