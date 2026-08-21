package models

type Vendor struct {
	AutoID      int     `json:"auto_id"`
	VendorID    string  `json:"vendor_id"`
	VendorName  string  `json:"vendor_name"`
	Address     *string `json:"address,omitempty"`
	Phone1      *string `json:"phone1,omitempty"`
	Phone2      *string `json:"phone2,omitempty"`
	IsActive    bool    `json:"is_active"`
	IDStatus    string  `json:"id_status"`
	UpdateBy    string  `json:"update_by"`
	UpdateDate  string  `json:"update_date"`
	IsDelete    bool    `json:"is_delete"`
}
