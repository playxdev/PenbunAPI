package models

type ProductGroup struct {
	AutoID           int     `json:"auto_id"`
	ProductGroupID   string  `json:"product_group_id"`
	ProductGroupName string  `json:"product_group_name"`
	Description      *string `json:"description,omitempty"`
	IsActive         bool    `json:"is_active"`
	UpdateBy         string  `json:"update_by"`
	UpdateDate       string  `json:"update_date"`
	IsDelete         bool    `json:"is_delete"`
}
