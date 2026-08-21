# ✅ PenbunAPI Development Checklist (v3.2.0)

> **สถานะปัจจุบัน:** v3.2.0 — สร้าง Backend ใหม่ทั้งหมดด้วย Go + Fiber + MSSQL  
> **เป้าหมาย:** ทำให้ API รองรับ Database Schema v2.1.0 100% พร้อม Unit Testing และ QA

---

## 🟢 Layer 1: System & Infrastructure (Foundation)

| สถานะ | รายการ | รายละเอียด |
|-------|--------|-----------|
| ✅ Done | **Fiber Server** | HTTP server ด้วย `github.com/gofiber/fiber/v2` พร้อม CORS + Recover middleware |
| ✅ Done | **Environment Config** | `config/env.go` — โหลดจาก `.env` มี default value ทุกตัว |
| ✅ Done | **Database Connection** | `config/database.go` — MSSQL connection pool (MaxOpenConns=25, MaxIdleConns=10) |
| ✅ Done | **JWT Authentication** | `middleware/jwt.go` — Bearer token validation + claims extraction (`username`, `user_level`) |
| ✅ Done | **Token Blacklist** | `config/blacklist.go` — `sync.RWMutex` protects in-memory blacklist map |
| ✅ Done | **Graceful Shutdown** | `main.go` — จับ `SIGINT`/`SIGTERM` ปิด server + log file |
| ✅ Done | **Transaction Audit Log** | `utils/transaction.go` — บันทึก TX start/step/commit/rollback พร้อม duration |
| ✅ Done | **Global Error Handler** | `middleware/error.go` — centralized error handler ผ่าน `fiber.Config.ErrorHandler` |
| ⬜ Pending | **Role/Permission** | ยังไม่มีระบบ Role-Based Access Control |

**Strukture โปรเจกต์:**
```
PenbunAPI/
├── main.go                    # Entry point, graceful shutdown, Fiber config
├── config/                    # env, database, logger, blacklist
├── controllers/               # 17 modules × 8 functions = 136 handlers
├── middleware/                 # JWT middleware + global error handler
├── models/                    # 16 entity structs + ApiResponse
├── routes/                    # public, v1 (protected), v2
├── utils/                     # transaction manager, response helpers
├── logs/                      # transaction.log (runtime)
├── .env                       # DB credentials, JWT secret
├── go.mod / go.sum
└── *_test.go                  # 109 tests across all packages
```

---

## 🟡 Layer 2: Independent Master Data (Configuration)

Master Data พื้นฐาน — ทุก module มี 8 ฟังก์ชันมาตรฐาน: Select All, Select By Paging, Select By ID, Select By Name (LIKE), Insert, Update By ID, Delete By ID (soft), Remove By ID (hard)

| สถานะ | Module | Table | Fields พิเศษ |
|-------|--------|-------|-------------|
| ✅ Done | **Unit Type** | `tb_unit_type` | `type_name`, `description` |
| ✅ Done | **Product Format Type** | `tb_product_format_type` | `type_name`, `description` |
| ✅ Done | **Vendor Type** | `tb_vendor_type` | `type_name`, `description` |
| ✅ Done | **Customer Type** | `tb_customer_type` | `type_name`, `base_credit_day`, `description` |
| ✅ Done | **Discount Type** | `tb_discount_type` | `type_name`, `description` |
| ✅ Done | **Product Category** | `tb_product_category` | `category_name`, `description` |
| ✅ Done | **Warehouse** | `tb_warehouse` | `warehouse_name`, `location` |

---

## 🟠 Layer 3: Dependent Master Data (Partners & Groups)

| สถานะ | Module | Table | Fields พิเศษ |
|-------|--------|-------|-------------|
| ✅ Done | **Vendor** | `tb_vendor` | `vendor_name`, `address`, `phone1`, `phone2` |
| ✅ Done | **Customer** | `tb_customer` | `customer_name`, `address`, `phone1`, `phone2`, `tax_id`, `credit_limit` |
| ✅ Done | **Discount** | `tb_discount` | `discount_name`, `is_percent`, `discount_value`, `min_order_amount`, `start_date`, `end_date` |
| ✅ Done | **Product Group** | `tb_product_group` | `group_name`, `description` |
| ✅ Done | **Publisher** (v1 compat) | `tb_publisher` | `publisher_name`, `address`, `phone` |
| ✅ Done | **Publisher Type** (v1 compat) | `tb_publisher_type` | `type_name`, `description` |
| ✅ Done | **Book** (v1 compat) | `tb_book` | `book_name`, `author`, `price` |
| ✅ Done | **Book Type** (v1 compat) | `tb_book_type` | `type_name`, `description` |

---

## 🔴 Layer 4: The Core Product (Hybrid System)

| สถานะ | Module | Table | หมายเหตุ |
|-------|--------|-------|---------|
| ✅ Done | **Product API** | `tb_product` | 8 functions, `count_stock`, Gen ID `PDT...` ใน Go โดยใช้ deterministic algorithm |
| ⬜ Pending | — | — | เมื่อ Product API เสร็จ ให้ลบ **Book API** และ **Book Type API** เดิม |

---

## 🟣 Layer 5: Inbound Transactions (Receive)

| สถานะ | Module | หมายเหตุ |
|-------|--------|---------|
| ⬜ Pending | **Receive Note API** (Header) | สร้างใบรับของ, Gen ID `RCV...` |
| ⬜ Pending | **Receive Item API** (Detail) | บันทึกรายการสินค้า, อัปเดต Stock (ถ้า `count_stock=1`) |

---

## 🔵 Layer 6: Outbound Transactions (Order)

| สถานะ | Module | หมายเหตุ |
|-------|--------|---------|
| ⬜ Pending | **Order API** (Header) | สร้างใบสั่งซื้อ, ตัด Credit Limit, Gen ID `ORD...` |
| ⬜ Pending | **Order Item API** (Detail) | ตรวจสอบ Stock, คำนวณส่วนลด, ตัด Stock จริง |

---

## 🧪 QA & Testing (Completed)

| สถานะ | รายการ | รายละเอียด |
|-------|--------|-----------|
| ✅ Done | **Unit Tests (109 tests)** | config(6) + controllers(43) + middleware(6) + models(5) + routes(41) + utils(8) |
| ✅ Done | **Race Detection** | `go test -race` — zero races |
| ✅ Done | **JWT Security Tests** | expired token, wrong secret, bad signature, missing header, malformed header, blacklisted token |
| ✅ Done | **Input Validation Tests** | 15 insert modules reject invalid JSON + empty required fields |
| ✅ Done | **Auth Enforcement Tests** | 54 routes verified to return 401 without valid JWT |
| ✅ Done | **Response Format Tests** | success(200), fail(400), error(500), login(200), unauthorized(401) |
| ✅ Done | **Model Serialization Tests** | JSON marshal/unmarshal, password field hidden in JSON output |
| ✅ Done | **Env Config Tests** | defaults + full override + partial override |
| ✅ Done | **Token Blacklist Tests** | add, check, multi-token, empty token, non-existent |
| ⬜ Pending | **Integration Tests** | `//go:build integration` — ต้องมี DB จริงถึง run ได้ |
| ⬜ Pending | **Postman Collection** | ยังไม่ได้ export Postman collection |

---

## 🚚 Additional Modules (Future)

| สถานะ | Module | หมายเหตุ |
|-------|--------|---------|
| ⬜ Pending | **Deliver Module** | ใบจัดส่งสินค้า + อัปเดตสถานะ |
| ⬜ Pending | **Return Module** | คืนสินค้า + ตรวจสอบสภาพ + คืนเงิน/เปลี่ยน |
| ⬜ Pending | **Invoice Module** | ใบแจ้งหนี้ + ติดตามสถานะการชำระเงิน |

---

## ⚡ Performance & High Load (Post-Implementation)

| สถานะ | รายการ |
|-------|--------|
| ⬜ Pending | Fiber Prefork + Concurrency tuning |
| ⬜ Pending | Connection Pool optimization (MaxOpenConns ≥200) |
| ⬜ Pending | Async logging / External log collector |
| ⬜ Pending | Redis caching for Master Data |
| ⬜ Pending | Gzip compression on select APIs |
| ⬜ Pending | Database index optimization |
| ⬜ Pending | Read Replica support |
| ⬜ Pending | Rate limiting |

---

## 📊 Summary

| Layer | สถานะ |
|-------|--------|
| Layer 1: System & Infrastructure | 8/9 ✅ (missing: RBAC) |
| Layer 2: Independent Master Data | 7/7 ✅ |
| Layer 3: Dependent Master Data | 8/8 ✅ |
| Layer 4: Product Core | 1/1 ✅ |
| Layer 5: Receive | 0/2 ⬜ |
| Layer 6: Order | 0/2 ⬜ |
| QA & Testing | 10/12 ✅ (missing: integration tests on real DB, Postman) |
| DB Schema Standardization | 2/2 ✅ (id_status→is_active rename, id_status NVARCHAR added to 4 tables) |

**Total API Endpoints:** 136 (17 modules × 8 functions)  
**Total Test Suites:** 109 tests, 0 failed, 0 races  
**Next Priority:** Integration Tests → Inbound Transaction (Receive) → RBAC
