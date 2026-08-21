# PenbunAPI v3.2.0

RESTful API for managing distribution and supply of books and stationery — built with Go + Fiber + MSSQL.

---

## Features

- **Authentication** – JWT login/logout with bcrypt password hashing + token blacklist
- **14 Master Data Modules** – 8 standard CRUD functions per module (112 endpoints)
- **Global Error Handler** – Centralized `middleware.GlobalErrorHandler` for consistent JSON error responses
- **Transaction Safety** – `utils.ExecuteTransaction()` with panic recovery + step logging
- **Partial Updates** – `COALESCE(NULLIF(...))` pattern for PATCH-like PUT
- **Soft Delete** – `is_delete = 1` with `update_by` tracking
- **Pagination** – `?page=&limit=` on all Select Page endpoints
- **LIKE Search** – `?name=` with `%LIKE%` pattern for Select By Name
- **Consistent Response** – `{status, message, data}` across all endpoints
- **Audit Logging** – Transaction steps logged to `logs/transaction.log`
- **Graceful Shutdown** – Safe server stop on SIGINT/SIGTERM
- **Testing** – 73 unit tests with race detection

---

## Quick Start

```bash
# Prerequisites: Go 1.26+, MSSQL Server

git clone <repo> && cd PenbunAPI
go mod tidy

# Configure .env
cp .env.example .env
# Edit DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, JWT_SECRET

go run main.go
# Server starts on :8089 (configurable via FIBER_PORT)
```

---

## Project Structure

```
PenbunAPI/
├── main.go                    # Entry point + Fiber config + graceful shutdown
├── config/
│   ├── env.go                 # Environment variable loading
│   ├── database.go            # MSSQL connection pool
│   ├── blacklist.go           # Thread-safe token blacklist
│   └── logger.go              # Transaction log file
├── middleware/
│   ├── jwt.go                 # JWT Bearer validation
│   └── error.go               # Global error handler
├── controllers/
│   ├── auth.go                # Login / Logout
│   ├── customer.go
│   ├── customerType.go
│   ├── vendor.go
│   ├── vendorType.go
│   ├── book.go
│   ├── bookType.go
│   ├── discount.go
│   ├── discountType.go
│   ├── productFormatType.go
│   ├── productCategory.go
│   ├── productGroup.go
│   ├── unitType.go
│   └── warehouse.go
├── models/
│   ├── api.go                 # ApiResponse struct
│   ├── user.go                # User + LoginRequest
│   └── ...                    # 14 entity structs
├── routes/
│   ├── public.go              # /api/v1/public/*
│   ├── v1.go                  # /api/v1/protected/*
│   └── v2.go                  # Placeholder for future
├── utils/
│   ├── transaction.go         # ExecuteTransaction with rollback
│   └── response.go            # JSON response helpers
├── logs/                      # Transaction audit logs
├── docs/                      # Documentation
├── .env                       # Environment variables
├── *_test.go                  # 73 tests
└── go.mod
```

---

## API Reference

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/public/login` | Login, returns JWT |
| POST | `/api/v1/public/logout` | Blacklist token (requires auth) |

### Master Data (all protected)

Each module has 8 endpoints under `/api/v1/protected/{module}`:

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/all` | Select all (is_delete = 0) |
| GET | `/page?page=1&limit=10` | Paginated results |
| GET | `/select/id/:id` | Select by code/ID |
| GET | `/select/name/:name` | LIKE search by name |
| POST | `/insert` | Create new record |
| PUT | `/update/:id` | Partial update |
| PUT | `/delete/:id?user=USER` | Soft delete |
| DELETE | `/remove/:id` | Hard delete |

#### Available Modules

| Module | Endpoint | Table |
|--------|----------|-------|
| Customer | `/customer/*` | `tb_customer` |
| Customer Type | `/customer-type/*` | `tb_customer_type` |
| Vendor | `/vendor/*` | `tb_vendor` |
| Vendor Type | `/vendor-type/*` | `tb_vendor_type` |
| Book | `/book/*` | `tb_book` |
| Book Type | `/book-type/*` | `tb_book_type` |
| Discount | `/discount/*` | `tb_discount` |
| Discount Type | `/discount-type/*` | `tb_discount_type` |
| Unit Type | `/unit-type/*` | `tb_unit_type` |
| Product Format Type | `/product-format-type/*` | `tb_product_format_type` |
| Product Category | `/product-category/*` | `tb_product_category` |
| Product Group | `/product-group/*` | `tb_product_group` |
| Warehouse | `/warehouse/*` | `tb_warehouse` |
| Product | `/product/*` | `tb_product` |

---

## Response Format

```json
{
  "status": "success | fail | error",
  "message": "Human-readable message",
  "data": { ... }
}
```

**Login response** additionally includes `"token": "jwt..."`.

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | `localhost` | MSSQL host |
| `DB_PORT` | `1433` | MSSQL port |
| `DB_USER` | `sa` | Database user |
| `DB_PASSWORD` | — | Database password |
| `DB_NAME` | `PENBUN` | Database name |
| `FIBER_PORT` | `8089` | Server port |
| `JWT_SECRET` | `default-secret` | JWT signing key |
| `LOG_FILE` | `logs/transaction.log` | Transaction log path |

---

## Database Schema (18 Tables)

All tables share universal audit columns:

| Column | Type | Default | Purpose |
|--------|------|---------|---------|
| `autoID` | `INT IDENTITY` | — | Primary key (not exposed in API) |
| `prefix` | `NVARCHAR(3)` | per table | Business ID prefix (e.g. `CUS`, `PDT`) |
| `is_active` | `BIT` | `1` | Technical: record enabled/disabled |
| `is_delete` | `BIT` | `0` | Soft delete flag |
| `id_status` | `NVARCHAR(20)` | `'ACTIVE'` | Business workflow state |
| `update_by` | `NVARCHAR(50)` | `'System'` | Operator username |
| `update_date` | `DATETIME` | `SYSDATETIMEOFFSET() AT TIME ZONE 'SE Asia Standard Time'` | Auto-set timestamp |

### Entity Tables & Columns

| # | Table | Specific Columns |
|---|-------|-----------------|
| 1 | `tb_book` | `book_id`, `book_name`, `author`, `price` |
| 2 | `tb_book_type` | `book_type_id`, `type_name`, `description` |
| 3 | `tb_company` | `company_id`, `company_code`, `name_th`, `name_en`, `description`, `tax_id`, `branch_code`, `contact_person`, `phone`, `mobile`, `fax`, `email`, `website`, `line_id`, `address`, `sub_district`, `district`, `province`, `zip_code`, `logo_url`, `vat_rate` |
| 4 | `tb_customer` | `customer_id`, `customer_type_id`, `customer_name`, `tax_id`, `branch_name`, `contact_person`, `phone1`, `phone2`, `email`, `line_id`, `address`, `sub_district`, `district`, `province`, `zip_code`, `credit_limit`, `credit_term_day`, `note` |
| 5 | `tb_customer_type` | `customer_type_id`, `type_name`, `description`, `base_credit_day` |
| 6 | `tb_discount` | `discount_id`, `discount_type_id`, `discount_name`, `discount_code`, `description`, `discount_value`, `is_percent`, `min_order_amount`, `start_date`, `end_date` |
| 7 | `tb_discount_type` | `discount_type_id`, `discount_type_name`, `description` |
| 8 | `tb_product` | `product_id`, `product_code`, `product_name`, `product_group_id`, `product_format_type_id`, `unit_type_id`, `vendor_id`, `count_stock`, `cost_price`, `sell_price`, `barcode`, `weight_kg`, `description` |
| 9 | `tb_product_category` | `product_category_id`, `category_name`, `category_code`, `description` |
| 10 | `tb_product_format_type` | `product_format_type_id`, `format_name`, `description` |
| 11 | `tb_product_group` | `product_group_id`, `product_category_id`, `product_group_name`, `description` |
| 12 | `tb_product_sku` | `sku_id`, `ref_product_id`, `barcode`, `vendor_part_no`, `variation_name`, `issue_no`, `volume_no`, `edition_label`, `cost_price`, `sell_price`, `description` |
| 13 | `tb_reference` | `ref_id` (PK), `ref_int`, `ref_text` |
| 14 | `tb_unit_type` | `unit_type_id`, `unit_type_name`, `description` |
| 15 | `tb_users` | `user_name` (unique), `user_password`, `user_level`, `user_id` |
| 16 | `tb_vendor` | `vendor_id`, `vendor_type_id`, `vendor_name`, `tax_id`, `branch_name`, `contact_person`, `phone1`, `phone2`, `email`, `website`, `address`, `sub_district`, `district`, `province`, `zip_code`, `credit_term_day`, `currency`, `note` |
| 17 | `tb_vendor_type` | `vendor_type_id`, `type_name`, `description` |
| 18 | `tb_warehouse` | `warehouse_id`, `warehouse_code`, `warehouse_name`, `description`, `is_main_dc`, `allow_negative_stock`, `location` (computed = `description`) |

---

## Testing

```bash
# Unit tests with race detection
go test -race -short ./...

# Coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

# Integration tests (requires live DB)
go test -tags=integration ./...
```

**Current:** 73 unit tests + 4 integration tests, 0 failed, 0 races across 6 packages.

---

## Libraries

| Library | Purpose |
|---------|---------|
| [Fiber v2](https://gofiber.io/) | Web framework |
| [go-mssqldb](https://github.com/denisenkom/go-mssqldb) | MSSQL driver |
| [Recover](https://docs.gofiber.io/api/middleware/recover) | Panic recovery middleware |
| [CORS](https://docs.gofiber.io/api/middleware/cors) | Cross-origin resource sharing |
| [golang-jwt v5](https://github.com/golang-jwt/jwt) | JWT auth |
| [bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt) | Password hashing |
| [godotenv](https://github.com/joho/godotenv) | .env loader |
| [testify](https://github.com/stretchr/testify) | Test assertions |

---

## Changelog

See [docs/CHANGELOG.md](docs/CHANGELOG.md).

## License

PENBUN License.
