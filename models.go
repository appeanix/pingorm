package pingorm

// Model sample entities here
type (
	TestTable struct {
		ID     uint32 `gorm:"primaryKey"`
		Field2 string
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
