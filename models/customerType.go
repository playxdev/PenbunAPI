package models

type CustomerType struct {
	AutoID         int     `json:"auto_id"`
	CustomerTypeID string  `json:"customer_type_id"`
	TypeName       string  `json:"type_name"`
	BaseCreditDay  *int    `json:"base_credit_day,omitempty"`
	Description    *string `json:"description,omitempty"`
	IsActive bool `json:"is_active"`
	UpdateBy       string  `json:"update_by"`
	UpdateDate     string  `json:"update_date"`
	IsDelete       bool    `json:"is_delete"`
}
