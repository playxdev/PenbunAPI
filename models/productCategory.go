package models

type ProductCategory struct {
	AutoID            int     `json:"auto_id"`
	ProductCategoryID string  `json:"product_category_id"`
	CategoryName      string  `json:"category_name"`
	Description       *string `json:"description,omitempty"`
	IsActive bool `json:"is_active"`
	UpdateBy          string  `json:"update_by"`
	UpdateDate        string  `json:"update_date"`
	IsDelete          bool    `json:"is_delete"`
}
