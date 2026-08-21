package models

type BookType struct {
	AutoID     int     `json:"auto_id"`
	BookTypeID string  `json:"book_type_id"`
	TypeName   string  `json:"type_name"`
	Description *string `json:"description,omitempty"`
	IsActive bool `json:"is_active"`
	UpdateBy   string  `json:"update_by"`
	UpdateDate string  `json:"update_date"`
	IsDelete   bool    `json:"is_delete"`
}
