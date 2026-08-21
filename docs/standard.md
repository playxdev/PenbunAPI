# 📏 PenbunAPI Development Standard

เอกสารนี้กำหนดแนวทางและมาตรฐานในการพัฒนา PenbunAPI อย่างเป็นระบบ เพื่อความสอดคล้อง รวดเร็ว และง่ายต่อการดูแลในระยะยาว

---

## 1. 🧱 Database Table Structure

- ทุกตารางต้องมี:
  - `autoID` (INT, Primary Key)
  - `prefix` (NVARCHAR(5)) สำหรับใช้สร้างรหัสแบบ dynamic
  - `[table]_id` หรือ `[table]_code` (NVARCHAR(50)) ใช้เป็นรหัสหลัก
  - `update_by`, `update_date`, `is_delete` (BIT DEFAULT 0)
- ตารางที่มี Business Workflow (transaction / master ที่ต้องการ state) ควรมี:
  - `id_status` (NVARCHAR(20) DEFAULT 'ACTIVE') — สถานะทางธุรกิจ เช่น `ACTIVE`, `CANCELLED`, `PENDING`, `EXPIRED`
- ตารางทั่วไป (lookup/reference) ใช้ `is_active` (BIT DEFAULT 1) สำหรับเปิด/ปิดการใช้งานเท่านั้น
- ใช้ Trigger:
  - `TRIG_AUTO_UPDATE_DATE_[TABLE_NAME]`
  - `TRIG_GENERATE_[TABLE]_ID` (ใช้ `prefix` เป็น dynamic ID)

---

## 2. 🧠 API Pattern (8 ฟังก์ชันหลัก)

1. Select All  
2. Select By Paging  
3. Select By ID  
4. Select By Name (LIKE `%name%`)  
5. Insert  
6. Update By ID  
7. Delete By ID (Soft Delete)  
8. Remove By ID (Hard Delete)  

---

## 3. 📦 API Design Guideline

- ใช้ `executeTransaction()` จาก `utils/transaction.go`
- ทุก Response ใช้ `models.ApiResponse` เท่านั้น
- ต้องมี field `status` เสมอ:
  - `"success"`: ทำงานสำเร็จ
  - `"fail"`: ข้อมูลผิดพลาด (Client Error)
  - `"error"`: ระบบขัดข้อง (Server Error)
  - `"unknow"`: ไม่ทราบสาเหตุ
- ทุก API Route อยู่ภายใต้ `/api/v1/protected/[module]`

---

## 4. 🔐 Authentication

- ใช้ JWT สำหรับ API ทั้งหมดที่ขึ้นต้นด้วย `/protected`
- Middleware ตรวจสอบ Token จาก `middleware/jwt.go`

---

## 5. 📄 Naming Convention

- ชื่อไฟล์: `camelCase`
- ชื่อตัวแปร/ฟังก์ชัน: `hungarian + camelCase`
- ชื่อ column/table: `snake_case`
- Prefix Code เช่น `B`, `C`, `P`, `V`

---

## 6. 🔄 Transaction Handling

- ทุก Insert, Update, Delete ต้องใช้ `executeTransaction()` เสมอ
- Rollback ถ้าเกิด panic/error เพื่อป้องกันข้อมูลเสียหาย

---

## 7. 🔍 LIKE Search (Select By Name)

```sql
WHERE type_name LIKE '%' + @type_name + '%' AND is_delete = 0
```
- ใช้ parameter `c.Params("name")`

---

## 8. 🧪 Data Validation

- Validate field ที่จำเป็นก่อน Insert/Update
- ตรวจสอบ foreign key ว่ามีอยู่จริง เช่น `publisher_type_id`, `customer_type_id`

---

## 9. 🌏 TimeZone & Logging

- ทุกตารางใช้เวลา `SE Asia Standard Time`
- บันทึก log ที่ `logs/transaction.log`

---

## 🧩 PenbunAPI Controller Template (8 Standard Functions)

Template นี้ใช้เป็นโครงสร้างพื้นฐานของ Controller สำหรับ Entity ทุกตัวในระบบ PenbunAPI

---

### 🔷 1. Select All

```go
func SelectAll<Entity>(c *fiber.Ctx) error {
    query := `SELECT ... FROM tb_<entity> WHERE is_delete = 0`
    rows, err := config.DB.Query(query)
    ...
}
```

---

### 🔷 2. Select By Paging

```go
func SelectPage<Entity>(c *fiber.Ctx) error {
    page := c.QueryInt("page", 1)
    limit := c.QueryInt("limit", 10)
    offset := (page - 1) * limit

    query := `SELECT ... FROM tb_<entity> WHERE is_delete = 0 ORDER BY update_date DESC OFFSET @Offset ROWS FETCH NEXT @Limit ROWS ONLY`
    ...
}
```

---

### 🔷 3. Select By ID

```go
func Select<Entity>ByID(c *fiber.Ctx) error {
    id := c.Params("id")
    query := `SELECT ... FROM tb_<entity> WHERE <entity>_id = @ID AND is_delete = 0`
    ...
}
```

---

### 🔷 4. Select By Name

```go
func Select<Entity>ByName(c *fiber.Ctx) error {
    name := c.Params("name")
    query := `SELECT ... FROM tb_<entity> WHERE type_name LIKE '%' + @Name + '%' AND is_delete = 0`
    ...
}
```

---

### 🔷 5. Insert

```go
func Insert<Entity>(c *fiber.Ctx) error {
    var item models.<Entity>
    if err := c.BodyParser(&item); err != nil {
        ...
    }

    query := `INSERT INTO tb_<entity> (..., id_status, ...) VALUES (..., COALESCE(NULLIF(?, ''), 'ACTIVE'), ...)`
    ...
}
```

---

### 🔷 6. Update By ID

```go
func Update<Entity>ByID(c *fiber.Ctx) error {
    id := c.Params("id")
    var item models.<Entity>
    if err := c.BodyParser(&item); err != nil {
        ...
    }

    query := `UPDATE tb_<entity> SET ..., id_status = COALESCE(NULLIF(?, ''), id_status) WHERE <entity>_id = @ID AND is_delete = 0`
    ...
}
```

---

### 🔷 7. Delete By ID (Soft Delete)

```go
func Delete<Entity>ByID(c *fiber.Ctx) error {
    id := c.Params("id")
    username := c.Query("user") // รับชื่อผู้ใช้จาก query string เช่น ?user=ROOT
    if username == "" {
        username = "UNKNOWN"
    }

    query := `UPDATE tb_<entity> SET is_delete = 1, update_by = @UpdateBy WHERE <entity>_id = @ID`
```

### 🔷 8. Remove By ID (Hard Delete)

```go
func Remove<Entity>ByID(c *fiber.Ctx) error {
    id := c.Params("id")
    query := `DELETE FROM tb_<entity> WHERE <entity>_id = @ID`
    ...
}
```

---

## 10. 🔍 Frontend Search Pattern

### **Filter locally (Client-side) vs Server-side**

ระบบมี 2 โหมดการค้นหา:

#### **✅ Filter locally (Client-side) - ค่าเริ่มต้น**
- 🔍 **Real-time filtering** - พิมพ์ทีละตัวอักษร ผลลัพธ์ขึ้นทันที
- 💻 **ทำงานบน Browser** - ไม่ส่ง request ไป server
- ⚡ **เร็วมาก** - ไม่ต้องรอ API response
- 📦 **ค้นหาเฉพาะข้อมูลที่โหลดมาแล้ว** - จำกัดอยู่ที่หน้าปัจจุบัน
- 🎯 **เหมาะสำหรับ:** ข้อมูลไม่เยอะ (< 100 records), ต้องการความเร็ว

**Implementation:**
```javascript
// ใช้ FilterableSearch module
this.filterSearch = new FilterableSearch(tableBodyId, searchInputId, {
    searchableColumns: [1, 2, 3],
    debounceDelay: 300,
    caseSensitive: false
});
```

#### **❌ Server-side (ปิด Filter locally)**
- 🔍 **Search on Enter** - กด Enter ถึงค้นหา
- 🌐 **ทำงานบน Server** - เรียก API `/select/name/{query}`
- 🐌 **ช้ากว่า** - ต้องรอ API response
- 📦 **ค้นหาทั้งหมดใน Database** - ค้นหาได้ทุก record
- 🎯 **เหมาะสำหรับ:** ข้อมูลเยอะมาก (> 1000 records), ต้องการค้นหาทั้ง database

**Implementation:**
```javascript
async handleSearch(event) {
    if (!this.useClientSideFilter && event.key === 'Enter') {
        const response = await apiClient.get(
            `${endpoint}/select/name/${encodeURIComponent(query)}`
        );
        // แสดงผลลัพธ์
    }
}
```

### **UI Component:**
```html
<div class="search-mode-toggle">
    <label>
        <input type="checkbox" id="clientSideFilter" checked>
        <span>Filter locally</span>
    </label>
</div>
```

### **แนะนำการใช้งาน:**

| สถานการณ์ | โหมดที่แนะนำ |
|-----------|--------------|
| ข้อมูลน้อย (< 100 records) | ✅ Filter locally |
| ต้องการความเร็ว | ✅ Filter locally |
| ข้อมูลเยอะ (> 1000 records) | ❌ Server-side |
| ต้องการค้นหาทั้ง database | ❌ Server-side |

---

🛠️ เปลี่ยน `<Entity>` เป็นชื่อ struct และ `<entity>` เป็นชื่อ table เช่น `VendorType`, `vendor_type`


---

## 11. 🏗️ Frontend Architecture (BaseTableManager)

เพื่อลดความซ้ำซ้อนของโค้ด (DRY Principle) และง่ายต่อการดูแลรักษา Frontend ให้ใช้มาตรฐาน **BaseTableManager** สำหรับหน้าจัดการข้อมูล (CRUD) ทุกหน้า

### **Concept**
- **BaseTableManager (Parent Class)**: จัดการ Logic กลาง เช่น Fetch Data, Pagination, Sorting, Search, Modal Toggle
- **Page Manager (Child Class)**: จัดการ Logic เฉพาะหน้า เช่น Form Fields, Validation, Custom Actions
- **Global Features**:
  - **Confirm Apply**: มี Checkbox ยืนยันก่อน Save/Update (Default: Enabled)
  - **Delete Confirmation**: Modal ยืนยันการลบพร้อม Checkbox

### **Implementation Pattern**

```javascript
// public/js/vendor.js
class VendorManager extends BaseTableManager {
    constructor() {
        super({
            endpoint: '/vendor',        // API Endpoint
            idField: 'vendor_code',     // Primary Key
            tableBodyId: 'vendorTableBody',
            paginationId: 'paginationControls',
            modalId: 'vendorModal',
            deleteModalId: 'deleteModal', // Shared Delete Modal
            formId: 'vendorForm',
            columns: [ ... ]            // Column definitions
        });
    }

    // Override: Custom Fetch Data (if response structure differs)
    async fetchData(page = 1) {
        // ...
    }
}

// Initialize
const vendorManager = new VendorManager();
window.vendorManager = vendorManager; // Expose to window for HTML onclick
```

### **Benefits**
1. **ลดโค้ด**: เขียนโค้ดเฉพาะส่วนที่ต่างกัน (ลดลง 80%)
2. **Consistency**: Pagination, Sorting, Search ทำงานเหมือนกันทุกหน้า
3. **Maintainability**: แก้ Logic กลางที่เดียว (BaseTableManager) มีผลทุกหน้า

---

## 12. 🌍 Frontend Configuration (env.js)

เพื่อให้สามารถ Deploy ใน Environment ที่ต่างกันได้ง่าย (Dev, Staging, Prod) โดยไม่ต้องแก้โค้ด ให้ใช้ไฟล์ `env.js` ในการเก็บค่า Configuration

### **Standard**
1. **File Location**: `public/env.js` (ควร exclude จาก git หรือใช้ `env.example.js`)
2. **Global Variable**: ใช้ `window.ENV` ในการเก็บค่า
3. **Fallback**: ในโค้ดต้องมีค่า Default เสมอถ้าไม่มี `env.js`

### **Example: public/env.js**
```javascript
window.ENV = {
    BASE_URL: 'http://localhost:8089/api/v1',
    TIMEOUT: 30000
};
```

### **Usage in config.js**
```javascript
const Config = {
    // Prioritize window.ENV, fallback to default
    BASE_URL: (window.ENV && window.ENV.BASE_URL) || 'http://localhost:3000/api/v1',
    TIMEOUT: (window.ENV && window.ENV.TIMEOUT) || 30000
};
```

---

## 13. 📌 Type Mapping Standard

- **Database (MSSQL)** -> **Go (Model)** -> **JSON**
- `BIT` -> `bool` -> `true/false` (e.g., `is_active`, `is_delete`)
- `NVARCHAR(20)` -> `string` -> `"text"` (e.g., `id_status`: `"ACTIVE"`, `"CANCELLED"`)
- `DATETIME` -> `string` (formatted) or `*time.Time` -> `"YYYY-MM-DDTHH:mm:ss..."`

---

> 🔖 เวอร์ชันล่าสุดของมาตรฐานนี้: v3.2.0 (Updated: 2026-06-14)

---

## 14. 🔄 Partial Update Pattern (Patch-like PUT)

เพื่อรองรับการแก้ไขข้อมูลบางส่วน (Partial Update) โดยที่ไม่ต้องส่งข้อมูลมาครบทุก field และไม่ให้ field ที่ไม่ได้ส่งมาถูกทับด้วยค่า Default value (เช่น `false`, `""`):

1. **Model Definition**:
   ใช้ **Pointer** (`*Type`) สำหรับ field ที่ต้องการรองรับ Optional/Partial Update โดยเฉพาะ `bool` และ `string` เพื่อให้แยกแยะระหว่าง `false/empty` กับ `nil` (missing) ได้
   ```go
   type Entity struct {
       Description *string `json:"description,omitempty"`
       IsActive    *bool   `json:"is_active,omitempty"`
   }
   ```

2. **Controller Logic (SQL Update)**:
   ใช้ฟังก์ชัน `COALESCE` (หรือ `ISNULL`) ใน SQL เพื่อเช็คว่าถ้า Parameter เป็น `NULL` ให้ใช้ค่าเดิมใน Database
   ```sql
   UPDATE tb_entity
   SET description = COALESCE(@Description, description),
       is_active = COALESCE(@IsActive, is_active)
   WHERE entity_id = @ID
   ```
   *หมายเหตุ: `COALESCE` จะคืนค่าแรกที่ไม่ใช่ NULL*

