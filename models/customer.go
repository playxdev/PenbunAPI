package models

type Customer struct {
	AutoID       int     `json:"auto_id"`
	CustomerID   string  `json:"customer_id"`
	CustomerName string  `json:"customer_name"`
	Address      *string `json:"address,omitempty"`
	Phone1       *string `json:"phone1,omitempty"`
	Phone2       *string `json:"phone2,omitempty"`
	TaxID        *string `json:"tax_id,omitempty"`
	CreditLimit  *float64 `json:"credit_limit,omitempty"`
	IsActive     bool    `json:"is_active"`
	IDStatus     string  `json:"id_status"`
	UpdateBy     string  `json:"update_by"`
	UpdateDate   string  `json:"update_date"`
	IsDelete     bool    `json:"is_delete"`
}
