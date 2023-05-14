package rio

import "time"

// Proto db model for proto
type Proto struct {
	ID        int64     `json:"id" yaml:"id"`
	Name      string    `json:"name" yaml:"name"`
	FileID    string    `json:"file_id" yaml:"file_id"`
	Methods   []string  `json:"methods" yaml:"methods" gorm:"serializer:json"`
	Types     []string  `json:"types" yaml:"types" gorm:"serializer:json"`
	CreatedAt time.Time `json:"created_at,omitempty" yaml:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty" yaml:"updated_at"`
}
