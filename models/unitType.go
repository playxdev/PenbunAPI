package models

type UnitType struct {
	AutoID       int     `json:"auto_id"`
	UnitTypeID   string  `json:"unit_type_id"`
	UnitTypeName string  `json:"unit_type_name"`
	Description  *string `json:"description,omitempty"`
	IsActive     bool    `json:"is_active"`
	UpdateBy     string  `json:"update_by"`
	UpdateDate   string  `json:"update_date"`
	IsDelete     bool    `json:"is_delete"`
}
