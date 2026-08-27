# PenbunAPI v4.0.0

ระบบจัดการศูนย์กระจายสินค้าหนังสือและเครื่องเขียน
Go 1.26 · Fiber v3.5.0 · MSSQL · JWT

---

## 1. หลักการออกแบบ

| | |
| :--- | :--- |
| **ฐานข้อมูลเป็นเจ้าของกฎ** | กฎที่บังคับได้ด้วย Trigger, Foreign Key, CHECK หรือ Stored Procedure ไม่เขียนซ้ำในโค้ด การเช็คซ้ำที่ไม่ตรงกันอันตรายกว่าการไม่เช็ค เพราะสร้างภาพลวงว่าปลอดภัย |
| **อ่านผ่าน View เขียนผ่าน Table/SP** | ทุก `SELECT` ยิงที่ View ซึ่งกรองแถวที่ถูกลบแล้ว การแตะสต็อกเรียก Stored Procedure เท่านั้น |
| **รหัสธุรกิจคือหน้าบ้าน** | ผู้เรียกเห็นและส่งแต่รหัสแบบ `CUSA000041` ส่วน `autoID` ที่ใช้ผูกความสัมพันธ์ถูกซ่อนไว้ในชั้น resolver |
| **ตัวตนมาจาก token** | `update_by` มาจาก JWT เสมอ ไม่เคยรับจาก query string หรือ body |
| **สิ่งที่ทำซ้ำได้ ต้องประกาศไม่ใช่เขียน** | master data 18 ตัวและเอกสาร 4 ชนิดเป็น descriptor ไม่ใช่ handler รายตัว |

---

## 2. โครงสร้าง

```
penbun-api/
├── Makefile
├── Dockerfile
├── go.mod
├── .env.example
├── .gitignore
├── .dockerignore                         กัน .env และ bin/ ไม่ให้เข้า build context
├── README.md
├── main.go                              ประกอบทุกชิ้นเข้าด้วยกัน + ปิดระบบอย่างเรียบร้อย
│
├── .github/workflows/ci.yml             format · layering · test · build
├── scripts/check-layering.sh             กฎการพึ่งพาระหว่างชั้น
├── docs/
│   └── DATABASE-CONTRACT.md              สัญญาที่ API พึ่งพาจากฐานข้อมูล
├── deploy/
│   ├── docker-compose.yml                MSSQL สำหรับพัฒนาบนเครื่อง
│   └── sql/                              สคริปต์ schema ที่ผ่านการรีวิว
├── test/integration/                     เทสต์ฐานข้อมูลจริง (`-tags=integration`)
│                                         การจองล็อกสต็อก · descriptor ตรงกับ schema
│
└── internal/                            Go บังคับเองว่า module ภายนอก import ไม่ได้
    │
    ├── config/                          ── ชั้นตั้งค่า
    │   ├── config.go                    อ่าน env + ตรวจความครบถ้วนแบบล้มทันที
    │   └── database.go                  connection pool (driver "sqlserver")
    │
    ├── schema/                          ── ชั้นคำอธิบายข้อมูล (ไม่รู้จัก HTTP verb และ SQL)
    │   ├── field.go                     Kind / Field / Ref / Filter
    │   ├── coerce.go                    แปลงและตรวจค่าจาก JSON
    │   └── args.go                      ตัวสะสมพารามิเตอร์ @pN
    │
    ├── platform/                        ── ชั้นพื้นฐาน (ไม่รู้จักธุรกิจ)
    │   ├── httpx/
    │   │   ├── envelope.go              รูปแบบ response เดียวของทั้งระบบ
    │   │   ├── errors.go                AppError + ตัวจัดการ error กลาง
    │   │   └── sqlerr.go                ตารางแปล error ของฐานข้อมูล → HTTP
    │   ├── logx/
    │   │   └── logx.go                  slog handler บรรทัดเดียว + โซนเวลา +7
    │   └── mw/
    │       ├── auth.go                  JWT · TokenStore · สิทธิ์ · เส้นทางสาธารณะ
    │       └── logger.go                access log หนึ่งบรรทัดต่อหนึ่งคำขอ
    │
    ├── repository/                      ── ชั้นเข้าถึงข้อมูล (ห้าม import fiber)
    │   ├── db.go                        WithTx · retry เมื่อ deadlock · AppLock
    │   ├── insert.go                    ที่อยู่เดียวของ INSERT ... OUTPUT ... INTO (ดูหัวข้อ 4.1)
    │   ├── scan.go                      แปลงชนิดข้อมูลให้พร้อมเป็น JSON
    │   └── resolver.go                  รหัสธุรกิจ → autoID + cache
    │
    ├── crud/                            ── เครื่องยนต์ที่ 1 : master data
    │   ├── resource.go                  descriptor
    │   ├── query.go                     ตัวประกอบ SQL
    │   ├── engine.go                    5 endpoint ต่อ resource
    │   └── query_test.go
    │
    ├── resources/                       ── descriptor 21 ตัว (ห้าม import fiber)
    │   ├── registry.go                  All() · Validate()
    │   ├── company.go                   บริษัท · ส่วนลด
    │   ├── lookup.go                    ตารางอ้างอิง 7 ตัว
    │   ├── master.go                    คลัง · กลุ่มสินค้า · คู่ค้า · ลูกค้า · สาย
    │   ├── product.go                   สินค้า · SKU · หนังสือ
    │   ├── user.go                      ผู้ใช้งาน — อ่านอย่างเดียว เฉพาะ ADMIN
    │   └── registry_test.go
    │
    └── domain/                          ── ส่วนที่มีตรรกะเฉพาะ
        ├── document/                    เครื่องยนต์ที่ 2 : เอกสาร
        │   ├── spec.go                  descriptor
        │   ├── repo.go                  header+items · batch insert · เรียก SP
        │   ├── handler.go               9 endpoint + วงจรสถานะ + จองล็อกก่อนโพสต์
        │   ├── specs.go                 spec ของเอกสารทั้ง 4 ชนิด
        │   └── specs_test.go
        ├── auth/
        │   ├── repo.go                  ตารางผู้ใช้ + ฟิลด์ควบคุมการเข้าสู่ระบบ
        │   ├── service.go               ล็อกบัญชี · บังคับเปลี่ยนรหัส · หมุน token
        │   └── handler.go
        ├── book/
        │   └── handler.go               เขียนสองตารางในทรานแซกชันเดียว
        ├── stock/
        │   ├── repo.go                  ทางเดียวที่โค้ดแตะสต็อกได้
        │   └── handler.go               คงเหลือ · เคลื่อนไหว · ปรับ · โอน · สร้างใหม่
        ├── allocation/
        │   └── handler.go               ประวัติการจัดส่ง + เสนอยอดงวดใหม่
        └── meta/
            └── handler.go               ค่า enum ที่อ่านจาก CHECK constraint จริง
```

**กฎที่บังคับด้วย CI:** `repository/`, `resources/` และ `schema/` ห้าม import `fiber`
ถ้าเผลอ import แปลว่าตรรกะของ HTTP รั่วลงชั้นข้อมูลแล้ว

> ข้อยกเว้นที่รู้ตัว: `schema` import `httpx` เพื่อคืน error ที่ระบุชื่อฟิลด์ได้
> เป็นการแลกความบริสุทธิ์ของชั้นกับคุณภาพของข้อความ error ซึ่งคุ้มสำหรับระบบที่มี
> ทางออกทาง HTTP ทางเดียว ถ้าวันหนึ่งต้องมีทางออกอื่น ให้แยก error ของ schema
> ออกมาเป็นชนิดของตัวเองแล้วให้ httpx เป็นคนแปล

---

## 3. เริ่มใช้งาน

```bash
cp .env.example .env
openssl rand -base64 48        # ใส่เป็น JWT_SECRET
# ตั้ง DB_PASSWORD และ CORS_ORIGINS ด้วย

make tidy
make run
```

process จะไม่เริ่มทำงานถ้า `JWT_SECRET`, `DB_PASSWORD` หรือ `CORS_ORIGINS` ไม่ได้ตั้ง
หรือ descriptor ตัวใดตัวหนึ่งไม่ถูกต้อง — ตั้งใจให้ล้มตั้งแต่ตอนเริ่ม ดีกว่าไปพังตอนมีคนใช้

```bash
make test        # เทสต์ที่ไม่ต้องใช้ฐานข้อมูล
make vet
make build
```

สำหรับ integration test ให้เริ่มฐานข้อมูลด้วย `docker compose -f deploy/docker-compose.yml up -d`
และตั้ง `PENBUN_INTEGRATION_DB_DSN` ก่อนรัน `make test-integration`.

### การนำขึ้นใช้งาน

รันบน DigitalOcean App Platform โดย build จาก `Dockerfile` ในราก ไม่ใช่ Go buildpack
เพราะ image ต้องมี `ca-certificates` สำหรับต่อฐานข้อมูลแบบเข้ารหัส และ `tzdata`
สำหรับแปลงวันที่ — buildpack ไม่ได้ให้มาทั้งคู่ ช่อง run command ต้องเว้นว่าง
เพราะ `ENTRYPOINT` เรียก binary ให้อยู่แล้ว

ตั้งค่าที่ต้องตรงกันสามจุด

| ตั้งที่ | ค่า | เหตุผล |
| :--- | :--- | :--- |
| HTTP port ของ component | `8089` | App Platform ส่งค่านี้มาทาง `PORT` ซึ่ง config อ่านก่อน `FIBER_PORT` ถ้าปล่อยเป็น 8080 จะไม่ตรงกับ `EXPOSE` และ `HEALTHCHECK` ใน Dockerfile |
| health check path | `/healthz` | ไม่ใช่ `/readyz` เพราะ `/readyz` ping ฐานข้อมูล ฐานล่มชั่วคราวจะกลายเป็นวนรีสตาร์ตที่ไม่ช่วยอะไร |
| จำนวน instance | `1` | รายการ token ที่ถูกเพิกถอนอยู่ในหน่วยความจำของ process ดูข้อ 11 |

ค่า env ที่ต้องตั้งบน App Platform เหมือน `.env.example` โดยทำ `DB_PASSWORD`
กับ `JWT_SECRET` เป็นชนิดเข้ารหัส และตั้ง `APP_ENV=production` เพื่อไม่ให้
ตารางเส้นทางถูกพิมพ์ลง log ตอนเริ่มทำงาน

`CORS_ORIGINS` ต้องมีโดเมนจริงของ PenbunWeb ไม่ใช่ค่า localhost ที่ติดมาจาก `.env`
คั่นหลายค่าด้วยจุลภาค และห้ามใส่ `*` — `config.go` ปฏิเสธตั้งแต่ตอนเริ่มทำงาน
เพราะ API ใช้ bearer token

```
CORS_ORIGINS=https://www.phenbun.com,https://phenbun.com,https://penbunweb-1kq.pages.dev,http://localhost:4173
```

โดเมน `.pages.dev` มีไว้เพื่อข้าม zone ของ Cloudflare ตอนไล่ปัญหา — มันคือ
deployment เดียวกันแต่ไม่มี zone คั่น จึงแยกได้ว่าอาการมาจากแอปหรือจากการตั้งค่า
zone ส่วน preview deployment ของ Pages ได้ subdomain สุ่มทุกครั้งและ CORS
ไม่รองรับ wildcard จึงยิง API ตัวจริงไม่ได้ ซึ่งตั้งใจให้เป็นแบบนั้น

ตั้งตัวแปรทั้งหมดไว้ **ที่เดียว** คือระดับ component เพราะปุ่ม *Add from .env*
เขียนลง component เสมอ การมีบางตัวที่ app-level ด้วยคือที่มาของหัวข้อถัดไป

**ตัวแปรระดับ component ทับระดับ app** บน App Platform ถ้า `CORS_ORIGINS` ถูก
import เข้าไปที่ component ตอนสร้างแอปจาก `.env` การไปแก้ที่หน้า App-Level
จะไม่มีผลใด ๆ และหน้าจอไม่ได้บอกว่าค่าไหนชนะ ต้องลบหรือแก้ที่ component

ค่าที่ process ถืออยู่จริงอ่านได้จากบรรทัดแรกของ Runtime Logs

```
20260826-14:13:17 | INFO | server starting | addr=:8089 env=production version=4.0.0 cors_origins=[https://www.phenbun.com]
```

และถ้ามีคำขอจาก origin ที่ไม่อยู่ในรายการ จะได้บรรทัดเตือนหนึ่งบรรทัดต่อหนึ่ง
origin ต่อการรันหนึ่งรอบ

```
20260826-14:13:18 | WARN | cors origin rejected | origin=https://www.phenbun.com method=GET path=/api/v2/receive-note allowed=[http://localhost:5173]
```

เขียนไว้เพราะการถูก CORS ปฏิเสธเป็นความล้มเหลวที่เงียบที่สุดในระบบ preflight
ตอบ `204` เหมือนกันทั้ง origin ที่อนุญาตและไม่อนุญาต ต่างกันแค่มี header
`Access-Control-Allow-Origin` หรือไม่ ส่วนเบราว์เซอร์บล็อกเองโดยไม่ส่งคำขอจริง
ตามมา ใน access log จึงเห็นแค่ `OPTIONS` ลอย ๆ และไม่มีอะไรบอกว่าเกิดอะไรขึ้น

อีกครึ่งหนึ่งของเรื่องนี้อยู่ฝั่ง PenbunWeb — `connect-src` ใน `public/_headers`
ต้องมี origin ของ API ด้วย เบราว์เซอร์บล็อกที่ CSP ก่อนจะถาม CORS เสียอีก
ตั้งถูกข้างเดียวยังยิงไม่ผ่าน

> App Platform ต่อฐานข้อมูลผ่านอินเทอร์เน็ตสาธารณะ ไม่ได้อยู่ใน VPC เดียวกับ Droplet
> และ IP ขาออกไม่คงที่ถ้าไม่ได้เปิด dedicated egress จึงตั้งกฎ firewall แคบ ๆ ไม่ได้
> `DB_ENCRYPT=true` กับ `DB_TRUST_CERT=false` เป็นขั้นต่ำที่ต้องมี

---

## 4. สองเรื่องที่ต้องอ่านก่อนแก้โค้ด

### 4.1 รหัสธุรกิจอ่านกลับจาก View และ `OUTPUT` ต้องมี `INTO` เสมอ

มีสองข้อผูกกันอยู่ ทั้งคู่มาจากข้อเท็จจริงเดียว คือ **ทุกตารางมี AFTER INSERT trigger**
(127 ตัวใน 32 ตาราง) ที่ทำงาน**หลัง**คำสั่ง `INSERT` จบ เพื่อเติมรหัสธุรกิจ

**ข้อแรก** ค่าที่ `OUTPUT` คืนเป็นค่า ณ ตอน `INSERT` ซึ่งรหัสธุรกิจยังว่าง จึงคืนได้แค่ `autoID`
แล้วต้องอ่านแถวกลับจาก View ในทรานแซกชันเดียวกัน

**ข้อสอง** SQL Server ปฏิเสธ `OUTPUT` ที่ไม่มี `INTO` บนตารางที่มี trigger เปิดอยู่

```
Msg 334 — The target table 'dbo.tb_vendor_type' of the DML statement cannot have
any enabled triggers if the statement contains an OUTPUT clause without INTO clause.
```

แปลว่า `OUTPUT INSERTED.autoID VALUES (...)` เฉย ๆ ใช้ไม่ได้กับตารางใดเลยในระบบนี้

```sql
-- ✅ ถูก
DECLARE @pb_inserted TABLE (autoID INT);
INSERT INTO dbo.tb_x (...) OUTPUT INSERTED.autoID INTO @pb_inserted VALUES (...);
SELECT autoID FROM @pb_inserted;              -- result set เดียวของ batch
-- แล้วอ่านแถวกลับ
SELECT * FROM vw_x WHERE x_auto = @autoID;    -- ในทรานแซกชันเดียวกัน

-- ❌ ผิด — Msg 334 ทุกตาราง
INSERT ... OUTPUT INSERTED.autoID VALUES (...)

-- ❌ ผิด — ได้ค่าว่างทุกครั้ง เพราะ trigger ยังไม่ทำงาน
INSERT ... OUTPUT INSERTED.customer_id INTO @t VALUES (...)
```

SQL ก้อนนี้มีที่อยู่เดียวคือ `repository.InsertReturningAuto(table, cols, placeholders)`
ทั้ง `crud.buildInsert`, `document.Repo.InsertHeader` และ `book.Handler.create` เรียกตัวนี้
ห้ามเขียน `INSERT ... OUTPUT` ด้วยมือที่อื่น

### 4.2 ต้องจองล็อกก่อนทุกครั้งที่แตะสต็อก

การตรวจว่าสต็อกพอ กับการหักสต็อกจริง เป็นคนละคำสั่งกัน
ถ้าเอกสารสองใบที่กินสินค้าตัวเดียวกันจากคลังเดียวกันเข้ามาพร้อมกัน ทั้งคู่จะอ่าน
ยอดคงเหลือค่าเดิม ผ่านการตรวจทั้งคู่ แล้วหักซ้อนกันจนติดลบทั้งที่คลังไม่ได้เปิด
`allow_negative_stock`

ทุกเส้นทางที่เรียก `stock.ApplyMovement` หรือ Stored Procedure ที่โพสต์เอกสาร
ต้องครอบด้วย

```go
repository.AppLock(ctx, tx, fmt.Sprintf("PENBUN:STOCK:%d:%d", skuAuto, whAuto), waitMS)
```

**จองเรียงตาม autoID จากน้อยไปมากเสมอ** ถ้าสองเส้นทางจองสลับลำดับกันจะเกิด deadlock
`Spec.LockPairsSQL` ทุกตัวจึงมี `ORDER BY` และมีเทสต์บังคับข้อนี้ไว้

> การล็อกฝั่งนี้ป้องกันได้เฉพาะทางเข้าผ่าน API เท่านั้น
> การเพิ่ม hint ล็อกในตัว Stored Procedure จะปลอดภัยกว่า เพราะครอบคลุมถึงคนที่
> เรียกกระบวนงานตรงจากเครื่องมือจัดการฐานข้อมูล

---

## 5. เพิ่มของใหม่

### เพิ่ม master resource

เขียน descriptor แล้วต่อท้าย `resources.All()` — ได้ครบ 5 endpoint ทันที
ไม่ต้องเขียน handler, SQL หรือเทสต์ของ CRUD เลย

```go
var Supplier = &crud.Resource{
	Name:          "supplier",
	Label:         "ซัพพลายเออร์",
	Source:        "dbo.vw_supplier",
	Table:         "tb_supplier",
	IDColumn:      "supplier_id",
	AutoColumn:    "supplier_auto",
	SearchColumns: []string{"supplier_name"},
	SortColumns:   map[string]string{"name": "supplier_name", "updated": "update_date"},
	DefaultSort:   "updated",
	Refs: []schema.Ref{
		{Field: "supplier_type_id", Table: "tb_supplier_type",
			Column: "ref_supplier_type_auto", Label: "ประเภท", Required: true},
	},
	Fields: []schema.Field{
		{Name: "supplier_name", Kind: schema.KindString, Required: true, MaxLen: 200, Label: "ชื่อ"},
	},
}
```

### จำกัดสิทธิ์ของ resource

`RequireLevel` ติดตัวกรองไว้ที่กลุ่มเส้นทางของ resource ทั้งกลุ่ม ไม่ใช่ทีละ endpoint
เส้นทางที่เพิ่มทีหลังจะได้ไม่หลุดออกไปเพราะมีคนลืมใส่

```go
ReadOnly:     true,
RequireLevel: []string{"ADMIN"},
```

`resources` ยังห้าม import `fiber` เหมือนเดิม descriptor จึงถือแค่ชื่อระดับสิทธิ์
เป็นข้อความ ส่วน `crud.Engine.Mount` เป็นฝ่ายแปลงเป็น `mw.RequireLevel` ตอนติดตั้ง

`user` เป็น resource เดียวที่ใช้ทั้งสองอย่างพร้อมกัน การสร้างและแก้ผู้ใช้ผ่าน generic
engine จะเปิดทางให้เขียน `user_password` กับ `user_level` ตรง ๆ ซึ่งต้องผ่าน bcrypt
และต้องมีกติกากันคนลบสิทธิ์ ADMIN คนสุดท้ายทิ้ง — งานนั้นเป็นของ domain แยกต่างหาก

### เพิ่มเอกสารชนิดใหม่

เขียน `document.Spec` แล้วต่อท้าย `document.All()` — ได้ครบ 9 endpoint
พร้อมวงจรสถานะและการจองล็อกก่อนโพสต์

`Spec.Validate()` จะปฏิเสธ spec ที่ไม่ได้ระบุ `LockPairsSQL` หรือ `TotalsSQL`
เพื่อไม่ให้เผลอสร้างเอกสารที่โพสต์ได้โดยไม่ล็อกหรือไม่คำนวณยอดรวมใหม่

---

## 6. แผนที่ endpoint

| กลุ่ม | เส้นทาง | จำนวน |
| :--- | :--- | ---: |
| การเข้าสู่ระบบ | `/auth/login` `/refresh` `/me` `/change-password` `/logout` · `/users/{id}/unlock` | 6 |
| Master data | 20 resource × 5 | 100 |
| ผู้ใช้งาน | `GET /user` `GET /user/{id}` — `RequireLevel: ADMIN` | 2 |
| เอกสาร | `receive-note` `order` `return-note` `vendor-return-note` × 9 | 36 |
| หนังสือ | `POST` `PUT` `DELETE /book` | 3 |
| สต็อก | `onhand` `movements` `adjust` `transfer` `rebuild` | 5 |
| ฝากขาย | `outstanding` `rebuild` | 2 |
| การจัดสรร | `history` `pull` | 2 |
| ระบบ | `/healthz` `/readyz` `/version` `/meta/enums` | 4 |
| | **รวม** | **160** |

---

## 7. รูปแบบ response

ทุกคำตอบใช้รูปแบบเดียวกันเสมอ ไม่มีข้อยกเว้น

```json
{
  "status": "success",
  "message": "พบข้อมูลลูกค้า 128 รายการ",
  "code": "OK",
  "data": { "items": [], "page": 1, "limit": 50, "total": 128, "total_pages": 3 },
  "trace_id": "3f9a2b71"
}
```

หน้าจอควรเขียนเงื่อนไขจาก `code` เท่านั้น **ห้ามอ่านจาก `message`** เพราะข้อความ
ปรับถ้อยคำได้ตลอดโดยไม่ถือเป็นการเปลี่ยนสัญญา

| `code` | HTTP | ความหมาย |
| :--- | ---: | :--- |
| `VALIDATION_FAILED` `FIELD_REQUIRED` `REF_NOT_FOUND` `INVALID_ENUM` `VALUE_OUT_OF_RANGE` | 400 | ผู้ใช้แก้ได้ ดู `errors[]` |
| `UNAUTHORIZED` `TOKEN_EXPIRED` | 401 | `TOKEN_EXPIRED` แปลว่าให้เรียก `/auth/refresh` ไม่ใช่เด้งออกไปหน้าเข้าสู่ระบบ |
| `FORBIDDEN` `MUST_CHANGE_PASSWORD` | 403 | |
| `NOT_FOUND` | 404 | |
| `DUPLICATE` `REF_IN_USE` `INSUFFICIENT_STOCK` `ALREADY_POSTED` | 409 | |
| `ENDPOINT_REMOVED` | 410 | เส้นทางรุ่นก่อน ให้ไปแก้โค้ดฝั่งผู้เรียก |
| `ACCOUNT_LOCKED` | 423 | |
| `BUSINESS_RULE` | 422 | `message` มาจากกฎในฐานข้อมูลโดยตรง แสดงให้ผู้ใช้อ่านได้เลย |
| `INTERNAL` `DB_UNAVAILABLE` | 500 · 503 | ใช้ `trace_id` ตามหาบรรทัด `ERROR` ใน log (ดูหัวข้อ 8) |

---

## 8. บันทึกการทำงาน

หนึ่งคำขอได้หนึ่งบรรทัด ทุกบรรทัดขึ้นต้นด้วยเวลารูปแบบเดียวกัน

```
20260824-21:15:04 | POST /auth/login | 401 | ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง
20260824-21:15:04 | GET /users | 403 | กรุณาเปลี่ยนรหัสผ่านก่อนใช้งานระบบ
20260824-21:15:04 | GET /auth/me | 200 | OK
```

| ช่อง | ที่มา |
| :--- | :--- |
| เวลา | `YYYYMMDD-HH:mm:ss` ที่ UTC+7 เสมอ ไม่ว่าเครื่องที่รันจะตั้งโซนเวลาอะไรไว้ |
| method + path | ตัด `/api/v2` ออกแล้ว เหลือเฉพาะส่วนที่ต่างกันจริง |
| status | รหัส HTTP |
| ข้อความ | `message` ใน response ถ้าไม่มีก็ใช้ข้อความมาตรฐานของ status code |

โซนเวลาตั้งด้วย `time.FixedZone` ไม่ใช่ `TimeZone: "Asia/Bangkok"` เพราะชื่อโซน
ต้องมี tzdata ติดมากับ image ด้วย ถ้าไม่มี Fiber จะตกกลับไปใช้เวลาของเครื่องเงียบ ๆ
ประเทศไทยไม่มี DST offset ตายตัวจึงถูกต้องตลอดปี

**4xx ไม่มีบรรทัดที่สอง** เพราะบรรทัดข้างบนบอกครบแล้ว ส่วน 5xx ได้บรรทัดเพิ่ม
เพื่อบันทึก error ต้นทางซึ่งไม่เคยถูกส่งออกไปหาผู้เรียก จึงไม่มีทางโผล่ใน access log

```
20260824-21:15:04 | POST /documents | 500 | ระบบขัดข้อง กรุณาลองใหม่อีกครั้ง
20260824-21:15:04 | ERROR | request failed | method=POST path=/api/v2/documents code=INTERNAL error="mssql: deadlock victim" trace_id=3f9a2b71
```

บรรทัดของ slog ใช้รูปแบบ `เวลา | ระดับ | ข้อความ | key=value ...` โดยใส่เครื่องหมาย
คำพูดเฉพาะค่าที่มีช่องว่างอยู่ข้างใน ระดับต่ำสุดที่พิมพ์ตั้งด้วย `LOG_LEVEL`
(`debug` `info` `warn` `error` ค่าเริ่มต้น `info`)

`trace_id` ยาว 8 ตัวอักษร ใช้จับคู่สิ่งที่ผู้ใช้แจ้งมากับบรรทัดใน log เท่านั้น
ไม่ได้เป็นคีย์ในฐานข้อมูลและไม่ต้องเดาไม่ได้

---

## 9. วงจรของเอกสาร

```
ใบรับสินค้า        DRAFT → CONFIRMED → POSTED                 → (CANCELLED)
ใบส่งหนังสือ       DRAFT → CONFIRMED → DELIVERED → INVOICED   → (CANCELLED)
ใบรับคืน           DRAFT → CONFIRMED → POSTED → CREDITED      → (CANCELLED)
ใบส่งคืนคู่ค้า      DRAFT → CONFIRMED → POSTED → SETTLED       → (CANCELLED)
```

* ใบส่งจบที่ `DELIVERED` ไม่ใช่ `POSTED` เหมือนอีกสามชนิด — หน้าจอต้องรู้ข้อนี้
* แก้รายการได้เฉพาะตอน `DRAFT`
* ยกเลิกได้เฉพาะ `DRAFT` และ `CONFIRMED` เอกสารที่โพสต์แล้วต้องออกเอกสารกลับรายการ
  เพราะบัญชีการเคลื่อนไหวสต็อกเป็นแบบเพิ่มอย่างเดียว
* ใบรับคืนกำหนดคลังปลายทางจากสภาพสินค้าเป็นรายบรรทัด ของชำรุดจะไม่ปนกลับเข้าคลังที่ใช้ขาย
* ใบส่งแบบฝากขาย **ต้องระบุ `period_key`** มิฉะนั้นจะไม่ถูกบันทึกลงประวัติ และการเสนอ
  ยอดของงวดถัดไปจะมองไม่เห็นรอบนี้ ซึ่งกว่าจะรู้ตัวคือเดือนถัดไปและแก้ย้อนหลังไม่ได้

---

## 10. เทสต์ที่ต้องมีก่อนขึ้นใช้งานจริง

รันบนฐานข้อมูลจริงในคอนเทนเนอร์ ข้อ 6 คือข้อที่จะ**ล้ม**ถ้าการจองล็อกหลุดไปจาก
เส้นทางใดเส้นทางหนึ่ง — ตั้งใจให้ล้ม

1. สร้างข้อมูลแล้วรหัสธุรกิจถูกเติมจริง ไม่ซ้ำ ไม่ข้าม
2. สร้างพร้อมกัน 50 เส้น → เลขต้องต่อเนื่อง
3. ลบแล้วแถวยังอยู่ · `is_delete=1` · `is_active=0` · `id_status='DELETED'`
4. ลบตารางแม่ที่ยังมีลูกอ้างอยู่ → `409 REF_IN_USE`
5. เดินครบวงจร รับเข้า → ส่งออก → รับคืน → ส่งคืนคู่ค้า แล้วเทียบยอดกับที่คำนวณมือ
6. **โพสต์สองใบพร้อมกันบนสินค้าและคลังเดียวกันที่สต็อกพอใบเดียว → สำเร็จ 1 ล้มเหลว 1**
7. โพสต์เกินสต็อก → `409 INSUFFICIENT_STOCK` และสต็อกต้องไม่ขยับเลย
8. โพสต์ซ้ำ → `409 ALREADY_POSTED`
9. สร้างยอดคงเหลือใหม่จากบัญชีการเคลื่อนไหว แล้วยอดต้องเท่าเดิมทุกแถว
10. descriptor ทุกตัวตรงกับ `INFORMATION_SCHEMA` — คอลัมน์มีจริง และ `MaxLen` เท่ากับความยาวคอลัมน์

ข้อ 10 อยู่ใน `test/integration/descriptor_drift_test.go` เขียนไว้เพราะ descriptor
ที่หลุดจาก schema ไม่แสดงอาการตอนเริ่มทำงานและไม่แสดงตอนอ่าน มันโผล่ตอนมีคนกดบันทึกจริง
`MaxLen` ที่กว้างกว่าคอลัมน์คือกรณีที่แย่ที่สุด เพราะผ่าน validation แล้วไปตายตอน `INSERT`
กลายเป็น 500 ทั้งที่ผู้ใช้กรอกมาถูกตามที่ API บอก

---

## 11. ข้อจำกัดที่รู้ตัว

| | |
| :--- | :--- |
| รายการ token ที่ถูกเพิกถอนเก็บในหน่วยความจำ | หายเมื่อรีสตาร์ต และไม่ทำงานข้าม instance รุ่นนี้จึงรองรับ instance เดียว — ต้องตั้งจำนวน instance บน App Platform เป็น 1 ไม่งั้น logout จะเพิกถอนได้แค่ instance ที่รับคำขอนั้น token เดิมยังใช้ที่ instance อื่นได้ต่อ ด้วยเหตุผลเดียวกันนี้ AppLock ก็ไม่ทำงานข้าม instance มี `TokenStore` เป็น interface เตรียมไว้เปลี่ยนแล้ว |
| ยังไม่มีกระบวนงานกลับรายการ | เอกสารที่โพสต์แล้วแก้ไม่ได้ ต้องให้ผู้ดูแลระบบใช้ `/stock/adjust` ไปก่อน |
| ต้องใช้ PenbunSQL v10 ขึ้นไป | ทุก resource และทุกเอกสารอ่านผ่าน View ตั้งแต่ v8 รันกับ v7 จะได้ `Invalid object name 'dbo.vw_...'` ตั้งแต่คำขอแรก · ตั้งแต่รุ่นนี้ descriptor ของ `company` และ `warehouse` เขียน `sub_district` / `district` / `zip_code` ซึ่งเป็นคอลัมน์ที่ v10 เพิ่มให้ `tb_warehouse` และเป็นคอลัมน์ที่ `vw_company` เพิ่งคืนมา รันกับ v9 สองหน้านี้จะพังที่ `Invalid column name` ส่วน resource อื่นยังทำงานปกติ |
| การควบคุมสิทธิ์ยังหยาบ | มีแค่ระดับผู้ดูแลกับผู้ใช้ทั่วไป รอตารางบทบาทและสิทธิ์ |
| รายการเอกสารใหญ่มากอาจช้า | กระบวนงานที่โพสต์เอกสารทำงานทีละรายการ ปรับได้ที่ฝั่งฐานข้อมูลเท่านั้น |

---

## 12. เอกสารที่เกี่ยวข้อง

| ไฟล์ | เนื้อหา |
| :--- | :--- |
| `docs/DATABASE-CONTRACT.md` | สิ่งที่ API พึ่งพาจากฐานข้อมูล และข้อเสนอถึงผู้ดูแล schema |
| `../PenbunSQL/README.md` | schema v10 · View · Stored Procedure · สมมติฐานทางธุรกิจ |
| `../PenbunSQL/SQL-STANDARD.md` | กฎการตั้งชื่อ · audit column · กฎของ trigger |
| `../PenbunWeb/README.md` | หน้าจอที่เรียก API นี้ และเครื่องยนต์ master data ฝั่งหน้าเว็บ |
| `../PENBUN-TODO.md` | งานที่เหลือของทั้งระบบ และตารางจุดที่สัญญายังไม่ตรงกัน |
