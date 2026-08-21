package models

type DiscountType struct {
	AutoID            int     `json:"auto_id"`
	DiscountTypeID    string  `json:"discount_type_id"`
	DiscountTypeName  string  `json:"discount_type_name"`
	Description       *string `json:"description,omitempty"`
	IsActive          bool    `json:"is_active"`
	UpdateBy          string  `json:"update_by"`
	UpdateDate        string  `json:"update_date"`
	IsDelete          bool    `json:"is_delete"`
}
