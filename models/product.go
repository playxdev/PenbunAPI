package models

type Product struct {
	AutoID             int      `json:"auto_id"`
	ProductID          *string  `json:"product_id,omitempty"`
	ProductCode        string   `json:"product_code"`
	ProductName        string   `json:"product_name"`
	ProductGroupID     string   `json:"product_group_id"`
	ProductFormatTypeID *string `json:"product_format_type_id,omitempty"`
	UnitTypeID         *string  `json:"unit_type_id,omitempty"`
	VendorID           *string  `json:"vendor_id,omitempty"`
	CountStock         bool     `json:"count_stock"`
	CostPrice          *float64 `json:"cost_price,omitempty"`
	SellPrice          *float64 `json:"sell_price,omitempty"`
	Barcode            *string  `json:"barcode,omitempty"`
	WeightKg           *float64 `json:"weight_kg,omitempty"`
	Description        *string  `json:"description,omitempty"`
	UpdateBy           string   `json:"update_by"`
	UpdateDate         string   `json:"update_date"`
	IsActive           bool     `json:"is_active"`
	IDStatus           string   `json:"id_status"`
	IsDelete           bool     `json:"is_delete"`
}
