package pingorm

import "time"

// Model sample entities here
type (
	Author struct {
		ID         uint32 `gorm:"primaryKey"`
		AuthorName string `json:"authorName"`
		Age        uint32 `json:"age"`
		Books      []Book `json:"books"`
	}
)

type (
	Editor struct {
		ID         uint32 `gorm:"primaryKey"`
		EditorName string `json:"editorName"`
		Age        uint32 `json:"age"`
		Books      []Book `json:"books"`
	}
)

type (
	Book struct {
		ID          uint32 `gorm:"primaryKey"`
		Title       string `json:"title"`
		PublishDate time.Time
		AuthorID    uint32
		EditorID    uint32
		Author      Author `gorm:"foreignKey:AuthorID;<-:false"`
		Editor      Editor `gorm:"foreignKey:EditorID;<-:false"`
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
