package models

type Discount struct {
	AutoID         int      `json:"auto_id"`
	DiscountID     string   `json:"discount_id"`
	DiscountName   string   `json:"discount_name"`
	IsPercent      *bool    `json:"is_percent,omitempty"`
	DiscountValue  *float64 `json:"discount_value,omitempty"`
	MinOrderAmount *float64 `json:"min_order_amount,omitempty"`
	StartDate      *string  `json:"start_date,omitempty"`
	EndDate        *string  `json:"end_date,omitempty"`
	IsActive       bool     `json:"is_active"`
	IDStatus       string   `json:"id_status"`
	UpdateBy       string   `json:"update_by"`
	UpdateDate     string   `json:"update_date"`
	IsDelete       bool     `json:"is_delete"`
}
