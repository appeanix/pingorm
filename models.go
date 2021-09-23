package pingorm

import "time"

// Model sample entities here
type (
	Author struct {
		ID    uint32 `gorm:"primaryKey"`
		Name  string
		Sex   string
		Dob   string
		Books []Book
	}
)

type (
	Editor struct {
		ID    uint32 `gorm:"primaryKey"`
		Name  string
		Sex   string
		Dob   string
		Books []Book
	}
)

type (
	Book struct {
		ID          uint32 `gorm:"primaryKey"`
		Title       string
		PublishDate time.Time
		AuthorID    uint32
		EditorID    uint32
		Author      Author `gorm:"foreignKey:AuthorID;"`
		Editor      Editor `gorm:"foreignKey:EditorID;"`
	}
)

type (
	QueryOption interface {
		GetSelectedFields() []string
		GetOmittedFields() []string
		GetPreloadedFields() []string
		HardDelete() bool
	}
)
