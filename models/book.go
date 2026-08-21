package models

type Book struct {
	AutoID      int     `json:"auto_id"`
	BookID      string  `json:"book_id"`
	BookName    string  `json:"book_name"`
	Author      *string `json:"author,omitempty"`
	Price       *float64 `json:"price,omitempty"`
	IsActive bool `json:"is_active"`
	UpdateBy    string  `json:"update_by"`
	UpdateDate  string  `json:"update_date"`
	IsDelete    bool    `json:"is_delete"`
}
