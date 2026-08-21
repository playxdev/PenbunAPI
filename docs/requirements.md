# Requirements Document

## Introduction

PenbunAPI v3.2.0 is a complete Go (Golang) RESTful backend API for a book and stationery distribution company (Penbun). It provides full CRUD operations for every database table in the PENBUN SQL Server database, secured with JWT authentication. The architecture follows a **Thin API** pattern: business rules, ID generation, and timestamps are delegated to DB triggers, while the API handles request routing, validation, transactional execution, and consistent JSON responses.

Every entity module exposes 8 standard endpoints. All mutating operations are wrapped in `executeTransaction()`. All responses conform to `models.ApiResponse`. Protected routes live under `/api/v1/protected/[module]` and require a valid JWT bearer token.

---

## Glossary

- **API**: The PenbunAPI v3.2.0 Go/Fiber application.
- **DB**: Microsoft SQL Server PENBUN database.
- **JWT**: JSON Web Token used for authentication.
- **Trigger**: SQL Server trigger that auto-generates IDs and sets `update_date`.
- **Soft Delete**: Setting `is_delete = 1` to logically remove a record without physical deletion.
- **Hard Delete**: Physical `DELETE FROM` statement that permanently removes a record.
- **COALESCE Update**: SQL pattern `SET col = COALESCE(NULLIF(@Param, ''), col)` used for partial updates.
- **executeTransaction**: Utility function in `utils/transaction.go` that wraps DB operations in a transaction with automatic rollback on error.
- **ApiResponse**: Universal response struct `{ status, message, data }` defined in `models/apiResponse.go`.
- **Prefix**: 3-character code (e.g., `CUS`, `PDT`) used by DB triggers to generate business IDs like `CUSA000001`.
- **Protected Route**: Any route under `/api/v1/protected/` that requires JWT middleware validation.
- **Public Route**: Any route under `/api/v1/public/` that does not require JWT.
- **Module**: A single database entity and its associated controller, model, and routes.
- **Paging**: Query-parameter-driven pagination using `?page=N&limit=N`.
- **Blacklist**: In-memory token blacklist used to invalidate JWTs on logout.
- **Update_by**: Operator/username field recorded on every mutation, supplied by the caller.
- **id_status / is_active**: BIT field (1 = Active, 0 = Inactive) for record status.
- **is_delete**: BIT field (0 = not deleted, 1 = soft-deleted).

---

## Requirements

---

### Requirement 1: Project Skeleton and Configuration

**User Story:** As a backend engineer, I want a well-structured Go project scaffold, so that all modules follow a consistent layout and the application starts cleanly.

#### Acceptance Criteria

1. THE API SHALL expose a root `main.go` that initialises Fiber, loads environment variables via `godotenv`, connects to the DB, registers all routes, and starts the HTTP server on the port defined in `.env`.
2. THE API SHALL gracefully shut down when an OS interrupt or SIGTERM signal is received, draining in-flight requests before exit.
3. THE Config SHALL load the following environment variables: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `JWT_SECRET`, `LOG_FILE` (or `TRANSACTION_LOG_FILE`), and `PORT`.
4. IF any required environment variable is missing at startup, THEN THE API SHALL log a descriptive error and exit with a non-zero status code.
5. THE DB Connection SHALL use the `go-mssqldb` driver with connection-string format `sqlserver://user:pass@host:port?database=name`.
6. THE Logger SHALL write transaction logs to the file path defined by `LOG_FILE`, creating the `logs/` directory if it does not exist.
7. THE API SHALL follow the project directory structure: `main.go`, `config/`, `controllers/`, `models/`, `routes/`, `middleware/`, `utils/`, `logs/`.
8. THE API SHALL use `camelCase` for file names, Hungarian-camelCase for Go identifiers, and `snake_case` for all SQL column/table references.

---

### Requirement 2: Authentication — Login and Logout

**User Story:** As a client application, I want to authenticate with username and password and receive a JWT token, so that I can access protected API resources.

#### Acceptance Criteria

1. THE API SHALL expose `POST /api/v1/public/login` as a public route.
2. WHEN a valid `user_name` and `user_password` are provided, THE Auth Controller SHALL verify the bcrypt password hash, generate a signed JWT, and return `{ "status": "success", "token": "<jwt>", "message": "Login successful" }`.
3. IF the username does not exist or the password is incorrect, THEN THE Auth Controller SHALL return `{ "status": "fail", "message": "Invalid credentials", "error": "..." }` with HTTP 401.
4. IF a database or server error occurs during login, THEN THE Auth Controller SHALL return `{ "status": "error", "message": "...", "error": "..." }` with HTTP 500.
5. THE JWT SHALL be signed with the `JWT_SECRET` from `.env` and include claims for `user_name` and `user_level`.
6. THE API SHALL expose `POST /api/v1/public/logout` as a public route.
7. WHEN a valid JWT is provided in the `Authorization: Bearer <token>` header at logout, THE Auth Controller SHALL add the token to the in-memory Blacklist and return `{ "status": "success", "message": "Logged out successfully" }`.
8. IF the token is missing or malformed at logout, THEN THE Auth Controller SHALL return `{ "status": "fail", "message": "No token provided" }` with HTTP 400.

---

### Requirement 3: JWT Middleware for Protected Routes

**User Story:** As a system, I want every protected endpoint to validate the JWT token, so that unauthenticated requests are rejected.

#### Acceptance Criteria

1. THE JWT Middleware SHALL validate the `Authorization: Bearer <token>` header on every request to `/api/v1/protected/`.
2. IF the token is absent, expired, or invalid, THEN THE JWT Middleware SHALL return HTTP 401 with `{ "status": "fail", "message": "Unauthorized" }`.
3. IF the token is present in the Blacklist, THEN THE JWT Middleware SHALL reject the request with HTTP 401.
4. WHEN the token is valid, THE JWT Middleware SHALL pass the request to the next handler.

---

### Requirement 4: Universal Response and Transaction Utility

**User Story:** As a developer, I want a shared response model and transaction wrapper, so that all endpoints return consistent JSON and all mutations are safely transactional.

#### Acceptance Criteria

1. THE ApiResponse Model SHALL be defined as `type ApiResponse struct { Status string \`json:"status"\`; Message string \`json:"message"\`; Data interface{} \`json:"data"\` }`.
2. THE executeTransaction Utility SHALL accept a `*sql.DB`, a list of SQL steps, and a logger; execute them inside a `BEGIN TRANSACTION`; commit on success; and roll back on any error.
3. WHEN a transaction step fails, THE executeTransaction Utility SHALL log the step index, error, duration, and rollback result to the transaction log.
4. WHEN a transaction succeeds, THE executeTransaction Utility SHALL log the total step count, duration, and commit result.
5. THE Transaction Log Format SHALL include: timestamp, step count, elapsed duration, and outcome (commit/rollback).

---

### Requirement 5: Company Module (`tb_company`)

**User Story:** As an admin, I want to manage the company profile (name, tax ID, address, VAT rate), so that the system stores the distributor's legal and contact information.

#### Acceptance Criteria

1. THE Company Controller SHALL expose `GET /api/v1/protected/company/select/all` returning all records `WHERE is_delete = 0`.
2. THE Company Controller SHALL expose `GET /api/v1/protected/company/select/page?page=N&limit=N` returning paginated records ordered by `update_date DESC`.
3. THE Company Controller SHALL expose `GET /api/v1/protected/company/select/id/:id` returning one record by `company_id`.
4. THE Company Controller SHALL expose `GET /api/v1/protected/company/select/name/:name` returning records where `name_th LIKE '%name%' OR name_en LIKE '%name%'` and `is_delete = 0`.
5. THE Company Controller SHALL expose `POST /api/v1/protected/company/insert` accepting fields: `prefix`, `company_code`, `name_th`, `name_en`, `description`, `tax_id`, `branch_code`, `contact_person`, `phone`, `mobile`, `fax`, `email`, `website`, `line_id`, `address`, `sub_district`, `district`, `province`, `zip_code`, `logo_url`, `vat_rate`, `update_by`, `is_active`.
6. WHEN a company record is inserted, THE Company Controller SHALL execute the insert inside `executeTransaction` and return `{ "status": "success", "message": "Company added successfully", "data": { "company_code": "..." } }`.
7. THE Company Controller SHALL expose `PUT /api/v1/protected/company/update/:id` using `COALESCE` partial-update SQL for all nullable/optional fields.
8. THE Company Controller SHALL expose `DELETE /api/v1/protected/company/delete/:id` performing a soft delete (`is_delete = 1, update_by = ?user`), accepting `?user=` query parameter.
9. THE Company Controller SHALL expose `DELETE /api/v1/protected/company/remove/:id` performing a hard delete (`DELETE FROM tb_company WHERE company_id = @ID`).
10. IF the company record is not found on select/update/delete, THEN THE Company Controller SHALL return `{ "status": "fail", "message": "Company not found" }` with HTTP 404.

---

### Requirement 6: Customer Type Module (`tb_customer_type`)

**User Story:** As an admin, I want to manage customer classification types (e.g., Walk-in, VIP, Corporate), so that customers can be categorised for pricing and credit terms.

#### Acceptance Criteria

1. THE CustomerType Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/customer-type/`.
2. THE CustomerType Controller SHALL use `customer_type_id` as the business key for select-by-ID, update, delete, and remove operations.
3. THE CustomerType Controller SELECT queries SHALL filter `WHERE is_delete = 0`.
4. THE CustomerType Controller SELECT BY NAME SHALL search `type_name LIKE '%name%' AND is_delete = 0`.
5. THE CustomerType Controller INSERT SHALL accept: `prefix`, `type_name`, `description`, `update_by`, `id_status`.
6. THE CustomerType Controller UPDATE SHALL use `COALESCE` to allow partial updates of `type_name`, `description`, `id_status`, `update_by`.
7. THE CustomerType Controller DELETE SHALL soft-delete by setting `is_delete = 1` and recording `update_by` from `?user=` query param.
8. THE CustomerType Controller REMOVE SHALL hard-delete by `DELETE FROM tb_customer_type WHERE customer_type_id = @ID`.

---

### Requirement 7: Customer Module (`tb_customer`)

**User Story:** As a sales operator, I want to manage customer records (contact details, credit limit, credit terms), so that orders can be linked to the correct customer.

#### Acceptance Criteria

1. THE Customer Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/customer/`.
2. THE Customer Controller SHALL use `customer_id` as the business key.
3. THE Customer Controller SELECT BY NAME SHALL search `customer_name LIKE '%name%' AND is_delete = 0`.
4. THE Customer Controller INSERT SHALL accept: `prefix`, `customer_type_id`, `customer_name`, `tax_id`, `branch_name`, `contact_person`, `phone1`, `phone2`, `email`, `line_id`, `address`, `sub_district`, `district`, `province`, `zip_code`, `credit_limit`, `credit_term_day`, `note`, `update_by`, `is_active`.
5. THE Customer Controller UPDATE SHALL use `COALESCE` partial update for all optional fields.
6. IF `customer_type_id` provided on insert does not exist in `tb_customer_type`, THEN THE Customer Controller SHALL return `{ "status": "fail", "message": "customer_type_id not found" }` with HTTP 400.
7. THE Customer Controller DELETE SHALL soft-delete with `is_delete = 1`.
8. THE Customer Controller REMOVE SHALL hard-delete.

---

### Requirement 8: Discount Type Module (`tb_discount_type`)

**User Story:** As a marketing manager, I want to manage discount categories (e.g., Percentage, Fixed Amount, Coupon), so that discounts can be classified and reported consistently.

#### Acceptance Criteria

1. THE DiscountType Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/discount-type/`.
2. THE DiscountType Controller SHALL use `discount_type_id` as the business key.
3. THE DiscountType Controller SELECT BY NAME SHALL search `discount_type_name LIKE '%name%' AND is_delete = 0`.
4. THE DiscountType Controller INSERT SHALL accept: `prefix`, `discount_type_name`, `description`, `update_by`, `is_active`.
5. THE DiscountType Controller UPDATE SHALL use `COALESCE` partial update.
6. THE DiscountType Controller DELETE SHALL soft-delete.
7. THE DiscountType Controller REMOVE SHALL hard-delete.

---

### Requirement 9: Discount Module (`tb_discount`)

**User Story:** As a marketing manager, I want to manage discount rules (value, percent flag, minimum order, validity dates), so that the correct discount is applied during order processing.

#### Acceptance Criteria

1. THE Discount Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/discount/`.
2. THE Discount Controller SHALL use `discount_id` as the business key.
3. THE Discount Controller SELECT BY NAME SHALL search `discount_name LIKE '%name%' AND is_delete = 0`.
4. THE Discount Controller INSERT SHALL accept: `prefix`, `discount_type_id`, `discount_name`, `discount_code`, `description`, `discount_value`, `is_percent`, `min_order_amount`, `start_date`, `end_date`, `update_by`, `is_active`.
5. THE Discount Controller UPDATE SHALL use `COALESCE` partial update for all nullable fields.
6. IF `discount_type_id` provided on insert does not exist in `tb_discount_type`, THEN THE Discount Controller SHALL return `{ "status": "fail", "message": "discount_type_id not found" }` with HTTP 400.
7. THE Discount Controller DELETE SHALL soft-delete.
8. THE Discount Controller REMOVE SHALL hard-delete.

---

### Requirement 10: Product Category Module (`tb_product_category`)

**User Story:** As an inventory manager, I want to manage top-level product categories (e.g., Books, Stationery), so that products are organised hierarchically.

#### Acceptance Criteria

1. THE ProductCategory Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/product-category/`.
2. THE ProductCategory Controller SHALL use `product_category_id` as the business key.
3. THE ProductCategory Controller SELECT BY NAME SHALL search `category_name LIKE '%name%' AND is_delete = 0`.
4. THE ProductCategory Controller INSERT SHALL accept: `prefix`, `category_name`, `category_code`, `description`, `update_by`, `is_active`.
5. THE ProductCategory Controller UPDATE SHALL use `COALESCE` partial update.
6. THE ProductCategory Controller DELETE SHALL soft-delete.
7. THE ProductCategory Controller REMOVE SHALL hard-delete.

---

### Requirement 11: Product Group Module (`tb_product_group`)

**User Story:** As an inventory manager, I want to manage product groups linked to categories (e.g., Fiction under Books), so that products have a two-level classification.

#### Acceptance Criteria

1. THE ProductGroup Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/product-group/`.
2. THE ProductGroup Controller SHALL use `product_group_id` as the business key.
3. THE ProductGroup Controller SELECT BY NAME SHALL search `product_group_name LIKE '%name%' AND is_delete = 0`.
4. THE ProductGroup Controller INSERT SHALL accept: `prefix`, `product_category_id`, `product_group_name`, `description`, `update_by`, `is_active`.
5. IF `product_category_id` provided on insert does not exist in `tb_product_category`, THEN THE ProductGroup Controller SHALL return `{ "status": "fail", "message": "product_category_id not found" }` with HTTP 400.
6. THE ProductGroup Controller UPDATE SHALL use `COALESCE` partial update.
7. THE ProductGroup Controller DELETE SHALL soft-delete.
8. THE ProductGroup Controller REMOVE SHALL hard-delete.

---

### Requirement 12: Product Format Type Module (`tb_product_format_type`)

**User Story:** As an inventory manager, I want to manage product format types (e.g., Paperback, Hardcover, Digital), so that products can be distinguished by physical format.

#### Acceptance Criteria

1. THE ProductFormatType Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/product-format-type/`.
2. THE ProductFormatType Controller SHALL use `product_format_type_id` as the business key.
3. THE ProductFormatType Controller SELECT BY NAME SHALL search `format_name LIKE '%name%' AND is_delete = 0`.
4. THE ProductFormatType Controller INSERT SHALL accept: `prefix`, `format_name`, `description`, `update_by`, `is_active`.
5. THE ProductFormatType Controller UPDATE SHALL use `COALESCE` partial update.
6. THE ProductFormatType Controller DELETE SHALL soft-delete.
7. THE ProductFormatType Controller REMOVE SHALL hard-delete.

---

### Requirement 13: Unit Type Module (`tb_unit_type`)

**User Story:** As an inventory manager, I want to manage unit-of-measure types (e.g., Piece, Pack, Box), so that product quantities are tracked in the correct unit.

#### Acceptance Criteria

1. THE UnitType Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/unit-type/`.
2. THE UnitType Controller SHALL use `unit_type_id` as the business key.
3. THE UnitType Controller SELECT BY NAME SHALL search `unit_type_name LIKE '%name%' AND is_delete = 0`.
4. THE UnitType Controller INSERT SHALL accept: `prefix`, `unit_type_name`, `description`, `update_by`, `is_active`.
5. THE UnitType Controller UPDATE SHALL use `COALESCE` partial update.
6. THE UnitType Controller DELETE SHALL soft-delete.
7. THE UnitType Controller REMOVE SHALL hard-delete.

---

### Requirement 14: Vendor Type Module (`tb_vendor_type`)

**User Story:** As a purchasing manager, I want to manage vendor classification types (e.g., Publisher, Distributor, Importer), so that vendors can be categorised for procurement reporting.

#### Acceptance Criteria

1. THE VendorType Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/vendor-type/`.
2. THE VendorType Controller SHALL use `vendor_type_id` as the business key.
3. THE VendorType Controller SELECT BY NAME SHALL search `type_name LIKE '%name%' AND is_delete = 0`.
4. THE VendorType Controller INSERT SHALL accept: `prefix`, `type_name`, `description`, `update_by`, `is_active`.
5. THE VendorType Controller UPDATE SHALL use `COALESCE` partial update.
6. THE VendorType Controller DELETE SHALL soft-delete.
7. THE VendorType Controller REMOVE SHALL hard-delete.

---

### Requirement 15: Vendor Module (`tb_vendor`)

**User Story:** As a purchasing manager, I want to manage vendor records (contact details, credit terms, currency), so that purchase orders can be sent to the correct supplier.

#### Acceptance Criteria

1. THE Vendor Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/vendor/`.
2. THE Vendor Controller SHALL use `vendor_id` as the business key.
3. THE Vendor Controller SELECT BY NAME SHALL search `vendor_name LIKE '%name%' AND is_delete = 0`.
4. THE Vendor Controller INSERT SHALL accept: `prefix`, `vendor_type_id`, `vendor_name`, `tax_id`, `branch_name`, `contact_person`, `phone1`, `phone2`, `email`, `website`, `address`, `sub_district`, `district`, `province`, `zip_code`, `credit_term_day`, `currency`, `note`, `update_by`, `is_active`.
5. IF `vendor_type_id` provided on insert does not exist in `tb_vendor_type`, THEN THE Vendor Controller SHALL return `{ "status": "fail", "message": "vendor_type_id not found" }` with HTTP 400.
6. THE Vendor Controller UPDATE SHALL use `COALESCE` partial update.
7. THE Vendor Controller DELETE SHALL soft-delete.
8. THE Vendor Controller REMOVE SHALL hard-delete.

---

### Requirement 16: Warehouse Module (`tb_warehouse`)

**User Story:** As a warehouse manager, I want to manage warehouse records (code, name, DC flag, negative stock flag), so that inventory locations are tracked per warehouse.

#### Acceptance Criteria

1. THE Warehouse Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/warehouse/`.
2. THE Warehouse Controller SHALL use `warehouse_id` as the business key.
3. THE Warehouse Controller SELECT BY NAME SHALL search `warehouse_name LIKE '%name%' AND is_delete = 0`.
4. THE Warehouse Controller INSERT SHALL accept: `prefix`, `warehouse_code`, `warehouse_name`, `description`, `is_main_dc`, `allow_negative_stock`, `update_by`, `is_active`.
5. THE Warehouse Controller UPDATE SHALL use `COALESCE` partial update.
6. THE Warehouse Controller DELETE SHALL soft-delete.
7. THE Warehouse Controller REMOVE SHALL hard-delete.

---

### Requirement 17: Product Module (`tb_product`)

**User Story:** As an inventory manager, I want to manage product master records (code, name, group, format, unit, vendor, prices), so that every product in the catalogue is uniquely identified and priced.

#### Acceptance Criteria

1. THE Product Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/product/`.
2. THE Product Controller SHALL use `product_id` as the business key.
3. THE Product Controller SELECT BY NAME SHALL search `product_name LIKE '%name%' AND is_delete = 0`.
4. THE Product Controller INSERT SHALL accept: `prefix`, `product_code`, `product_name`, `product_group_id`, `product_format_type_id`, `unit_type_id`, `vendor_id`, `count_stock`, `cost_price`, `sell_price`, `barcode`, `weight_kg`, `description`, `update_by`, `is_active`.
5. IF `product_group_id` provided on insert does not exist in `tb_product_group`, THEN THE Product Controller SHALL return `{ "status": "fail", "message": "product_group_id not found" }` with HTTP 400.
6. THE Product Controller UPDATE SHALL use `COALESCE` partial update for all nullable fields.
7. THE Product Controller DELETE SHALL soft-delete.
8. THE Product Controller REMOVE SHALL hard-delete.

---

### Requirement 18: Product SKU Module (`tb_product_sku`)

**User Story:** As an inventory manager, I want to manage product SKU variants (barcode, issue/volume/edition, cost/sell price), so that multiple variants of the same product can be stocked and sold separately.

#### Acceptance Criteria

1. THE ProductSKU Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/product-sku/`.
2. THE ProductSKU Controller SHALL use `sku_id` as the business key.
3. THE ProductSKU Controller SELECT BY NAME SHALL search `variation_name LIKE '%name%' AND is_delete = 0`.
4. THE ProductSKU Controller INSERT SHALL accept: `prefix`, `ref_product_id`, `barcode`, `vendor_part_no`, `variation_name`, `issue_no`, `volume_no`, `edition_label`, `cost_price`, `sell_price`, `description`, `update_by`, `is_active`.
5. IF `ref_product_id` provided on insert does not exist in `tb_product`, THEN THE ProductSKU Controller SHALL return `{ "status": "fail", "message": "ref_product_id not found" }` with HTTP 400.
6. THE ProductSKU Controller UPDATE SHALL use `COALESCE` partial update.
7. THE ProductSKU Controller DELETE SHALL soft-delete.
8. THE ProductSKU Controller REMOVE SHALL hard-delete.

---

### Requirement 19: Reference Module (`tb_reference`)

**User Story:** As a system administrator, I want to manage generic key-value reference records (ref_id, ref_int, ref_text), so that application-wide configuration values are stored centrally.

#### Acceptance Criteria

1. THE Reference Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/reference/`.
2. THE Reference Controller SHALL use `ref_id` (VARCHAR, not auto-generated) as the primary and business key.
3. THE Reference Controller SELECT queries SHALL filter `WHERE is_delete = 0` (treating NULL as 0).
4. THE Reference Controller SELECT BY NAME SHALL search `ref_id LIKE '%name%' AND (is_delete = 0 OR is_delete IS NULL)`.
5. THE Reference Controller INSERT SHALL accept: `ref_id`, `ref_int`, `ref_text`, `update_by`, `prefix`, `id_status`.
6. THE Reference Controller UPDATE SHALL use `COALESCE` partial update for `ref_int`, `ref_text`, `update_by`, `id_status`.
7. THE Reference Controller DELETE SHALL soft-delete by `UPDATE tb_reference SET is_delete = 1 WHERE ref_id = @ID`.
8. THE Reference Controller REMOVE SHALL hard-delete by `DELETE FROM tb_reference WHERE ref_id = @ID`.

---

### Requirement 20: User Management Module (`tb_users`)

**User Story:** As an admin, I want to manage user accounts (username, hashed password, level), so that system access is controlled per user role.

#### Acceptance Criteria

1. THE User Controller SHALL expose all 8 standard endpoints under `/api/v1/protected/users/`.
2. THE User Controller SHALL use `user_id` as the business key for ID-based operations.
3. THE User Controller SELECT ALL and SELECT BY PAGING SHALL return all fields **except** `user_password`.
4. THE User Controller SELECT BY NAME SHALL search `user_name LIKE '%name%' AND (is_delete = 0 OR is_delete IS NULL)`.
5. THE User Controller INSERT SHALL accept `user_name`, `user_password` (plaintext), `user_level`, `prefix`, `update_by`, `id_status`; THE User Controller SHALL hash the plaintext password with bcrypt before inserting.
6. THE User Controller UPDATE SHALL allow partial update of `user_name`, `user_level`, `id_status`, `update_by`; IF `user_password` is provided, THE User Controller SHALL re-hash it with bcrypt before updating.
7. THE User Controller DELETE SHALL soft-delete by `is_delete = 1`.
8. THE User Controller REMOVE SHALL hard-delete.
9. THE User Controller SHALL never return `user_password` in any response payload.

---

### Requirement 21: Route Registration

**User Story:** As a developer, I want all routes registered from dedicated route files, so that route definitions are separated from controller logic and easy to audit.

#### Acceptance Criteria

1. THE Routes Package SHALL contain `public.go` registering `/api/v1/public/login` and `/api/v1/public/logout`.
2. THE Routes Package SHALL contain `v1.go` registering all 8 endpoints for each of the 15 entity modules under `/api/v1/protected/`, wrapped with the JWT middleware group.
3. WHEN a request is made to an undefined route, THE API SHALL return HTTP 404 with `{ "status": "fail", "message": "Route not found" }`.
4. THE Routes Package SHALL group protected routes so that the JWT middleware is applied once at the group level, not per-endpoint.

---

### Requirement 22: Error Handling and Input Validation

**User Story:** As a developer, I want consistent error responses and input validation across all endpoints, so that clients receive actionable feedback on bad requests.

#### Acceptance Criteria

1. WHEN a request body cannot be parsed (invalid JSON), THE API SHALL return HTTP 400 with `{ "status": "fail", "message": "Invalid request body" }`.
2. WHEN a required field is missing or empty on insert, THE API SHALL return HTTP 400 with `{ "status": "fail", "message": "<field> is required" }`.
3. WHEN a DB query returns an error (not a "no rows" case), THE API SHALL return HTTP 500 with `{ "status": "error", "message": "Database error: ..." }`.
4. WHEN `sql.ErrNoRows` is returned on a select-by-ID, THE API SHALL return HTTP 404 with `{ "status": "fail", "message": "<entity> not found" }`.
5. IF an unhandled panic occurs inside a request handler, THE API SHALL recover and return HTTP 500 with `{ "status": "error", "message": "Internal server error" }`.

---

### Requirement 23: Paging Behaviour

**User Story:** As a client application, I want consistent and predictable paging across all modules, so that large record sets can be retrieved incrementally.

#### Acceptance Criteria

1. WHEN `?page=N&limit=N` are provided, THE API SHALL return records from offset `(page-1)*limit` with a maximum of `limit` rows, ordered by `update_date DESC`.
2. WHEN `page` or `limit` are absent, THE API SHALL default `page` to `1` and `limit` to `10`.
3. IF `page` or `limit` are non-integer or less than 1, THEN THE API SHALL return HTTP 400 with `{ "status": "fail", "message": "Invalid page or limit parameter" }`.
4. THE Paging Response SHALL include `total_records`, `page`, `limit`, and `data` array in the `data` field of `ApiResponse`.

---

### Requirement 24: Soft Delete Behaviour

**User Story:** As an operator, I want soft-deleted records to be hidden from all standard queries but recoverable via hard delete, so that data is not permanently lost by mistake.

#### Acceptance Criteria

1. THE API SHALL set `is_delete = 1` and record `update_by` (from `?user=` query param, defaulting to `"UNKNOWN"` if absent) for all soft-delete operations.
2. WHEN a soft delete is performed, THE API SHALL execute the update inside `executeTransaction`.
3. ALL Select All and Select By Paging queries SHALL include `WHERE is_delete = 0` to exclude soft-deleted records.
4. ALL Select By ID and Select By Name queries SHALL include `AND is_delete = 0` to exclude soft-deleted records.
5. WHEN `?user=` query param is absent on delete, THE API SHALL default `update_by` to `"UNKNOWN"`.

---

### Requirement 25: Logging Standard

**User Story:** As an operations engineer, I want structured transaction logs written to a file, so that every DB mutation can be audited and diagnosed.

#### Acceptance Criteria

1. THE Logger SHALL write to the file path specified by `LOG_FILE` env variable.
2. WHEN the log directory does not exist at startup, THE Logger SHALL create it automatically.
3. WHEN `executeTransaction` completes (success or failure), THE Logger SHALL write a log entry containing: UTC timestamp, operation name, step count, total duration in milliseconds, and outcome (`COMMIT` or `ROLLBACK`).
4. THE Log Entry Format SHALL be human-readable plain text on a single line per transaction.
5. THE Logger SHALL not write sensitive data (passwords, JWT tokens) to the log file.
