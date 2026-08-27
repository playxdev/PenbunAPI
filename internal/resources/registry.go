// Path: internal/resources/registry.go
package resources

import "penbun/api/internal/crud"

// All คืน resource ทุกตัวที่ติดตั้งผ่าน generic CRUD engine
//
// เรียงตามลำดับชั้นของข้อมูล จากตารางที่ไม่อ้างใครเลยไปหาตารางที่อ้างคนอื่นมากที่สุด
// เพื่อให้อ่านคู่กับเอกสารโครงสร้างฐานข้อมูลได้ตรงกัน
//
// เพิ่ม resource ใหม่ = เขียน descriptor แล้วต่อท้ายรายการนี้
func All() []*crud.Resource {
	return []*crud.Resource{
		// ชั้นระบบ — อ่านอย่างเดียว และเฉพาะ ADMIN
		User,

		// ชั้นข้อมูลอ้างอิง
		Company,
		CustomerType,
		VendorType,
		DiscountType,
		DiscountGroup,
		ProductCategory,
		ProductFormatType,
		UnitType,
		BookType,

		// ชั้นข้อมูลหลัก
		Warehouse,
		ProductGroup,
		Vendor,
		Customer,
		Discount,

		// ชั้นสายจัดจำหน่าย
		Route,
		CustomerRoute,

		// ชั้นสินค้า
		Product,
		ProductSKU,
		Book, // อ่านอย่างเดียว — การเขียนอยู่ที่ domain/book

		// ชั้นกฎราคา — อ้างทั้งกลุ่มส่วนลด ลูกค้า สาย และ SKU จึงอยู่ท้ายสุด
		PriceRule,
	}
}

// Validate ตรวจ descriptor ทุกตัวตอน process start
// descriptor ที่ผิดควรทำให้ start ไม่ขึ้น ดีกว่าปล่อยให้พังตอนมีคนใช้งานจริง
func Validate() error {
	for _, r := range All() {
		if err := r.Validate(); err != nil {
			return err
		}
	}
	return nil
}
