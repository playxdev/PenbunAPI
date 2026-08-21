package models

type Warehouse struct {
	AutoID       int     `json:"auto_id"`
	WarehouseID  string  `json:"warehouse_id"`
	WarehouseName string `json:"warehouse_name"`
	Location     *string `json:"location,omitempty"`
	IsActive bool `json:"is_active"`
	UpdateBy     string  `json:"update_by"`
	UpdateDate   string  `json:"update_date"`
	IsDelete     bool    `json:"is_delete"`
}
