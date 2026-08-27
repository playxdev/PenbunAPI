// Path: main.go
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/joho/godotenv"

	"penbun/api/internal/config"
	"penbun/api/internal/crud"
	"penbun/api/internal/domain/allocation"
	"penbun/api/internal/domain/auth"
	"penbun/api/internal/domain/book"
	"penbun/api/internal/domain/document"
	"penbun/api/internal/domain/meta"
	"penbun/api/internal/domain/stock"
	"penbun/api/internal/domain/user"
	"penbun/api/internal/platform/httpx"
	"penbun/api/internal/platform/logx"
	"penbun/api/internal/platform/mw"
	"penbun/api/internal/repository"
	"penbun/api/internal/resources"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("startup failed: %v", err)
	}
}

func run() error {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	lg := slog.New(logx.NewHandler(os.Stdout, parseLevel(cfg.LogLevel)))
	slog.SetDefault(lg)

	// ตรวจ descriptor ทั้งหมดก่อนต่อฐานข้อมูล
	// descriptor ที่ผิดต้องทำให้ start ไม่ขึ้น ดีกว่าไปพังตอนมีคนใช้งาน
	if err := resources.Validate(); err != nil {
		return err
	}
	if err := document.Validate(); err != nil {
		return err
	}

	ctx := context.Background()
	sqlDB, err := config.OpenDB(ctx, cfg)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	db := repository.New(sqlDB, lg)
	resolver := repository.NewResolver(db, 5*time.Minute)

	app := buildApp(cfg, lg)
	registerOps(app, db, cfg)
	registerLegacyGone(app)

	authn := &mw.Authenticator{
		Secret: []byte(cfg.JWTSecret),
		Store:  mw.NewMemoryTokenStore(),
	}

	// ทุกเส้นทางของ API อยู่ใต้ middleware ชุดเดียวกัน
	// ข้อยกเว้นเรื่องสิทธิ์กำหนดไว้เป็นรายการชัดเจนใน package mw ไม่ใช่ด้วยลำดับ
	// การลงทะเบียน เพื่อไม่ให้เส้นทางหลุดเป็นสาธารณะโดยบังเอิญเมื่อมีคนสลับบรรทัด
	api := app.Group("/api/v2", authn.Protect(), mw.RequirePasswordChanged())

	crudEngine := crud.NewEngine(db, resolver)
	docEngine := document.NewEngine(db, resolver, crudEngine, cfg)

	auth.NewHandler(auth.NewService(auth.NewRepo(db), cfg, authn.Store), authn).Register(api)

	if err := crudEngine.MountAll(api, resources.All()); err != nil {
		return err
	}
	if err := docEngine.MountAll(api, document.All()); err != nil {
		return err
	}

	book.NewHandler(db, resolver, crudEngine).Register(api)
	user.NewHandler(db, resolver, crudEngine, cfg).Register(api)
	stock.NewHandler(stock.NewRepo(db), db, resolver, cfg).Register(api)
	allocation.NewHandler(db, resolver).Register(api)
	meta.NewHandler(db).Register(api)

	return serve(app, cfg, lg)
}

func buildApp(cfg *config.Config, lg *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		ServerHeader:  "PENBUN",
		AppName:       "PenbunAPI v" + cfg.Version,
		CaseSensitive: true,
		ReadTimeout:   cfg.ReadTimeout,
		WriteTimeout:  cfg.WriteTimeout,
		IdleTimeout:   cfg.IdleTimeout,
		BodyLimit:     cfg.BodyLimit,
		ErrorHandler:  httpx.GlobalErrorHandler(lg),
	})

	// requestid ต้องมาก่อนทุกตัว เพราะ logger และ error handler อ้าง trace_id
	app.Use(requestid.New(requestid.Config{Generator: shortID}))
	app.Use(recover.New())
	app.Use(mw.AccessLog())
	// ต้องมาก่อน cors.New เพื่อให้เห็น origin ก่อนที่ middleware จะตัดสิน
	app.Use(mw.CORSAudit(cfg.CORSOrigins, lg))
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORSOrigins,
		AllowMethods: []string{
			fiber.MethodGet, fiber.MethodPost, fiber.MethodPut,
			fiber.MethodDelete, fiber.MethodOptions,
		},
		AllowHeaders:     []string{fiber.HeaderAuthorization, fiber.HeaderContentType},
		AllowCredentials: false,
	}))

	return app
}

func serve(app *fiber.App, cfg *config.Config, lg *slog.Logger) error {
	app.Hooks().OnPreShutdown(func() error {
		lg.Info("shutting down")
		return nil
	})
	app.Hooks().OnPostShutdown(func(err error) error {
		if err != nil {
			lg.Error("shutdown error", "error", err)
		} else {
			lg.Info("server stopped cleanly")
		}
		return nil
	})

	// Listen ต้องอยู่ใน goroutine ไม่งั้นจะไม่มีใครรอสัญญาณปิดและ hook จะไม่ทำงาน
	go func() {
		addr := ":" + cfg.Port
		// พิมพ์รายการ CORS ที่ process นี้ถืออยู่จริง ไม่ใช่ที่ตั้งใจตั้งไว้
		//
		// บนแพลตฟอร์มที่ตัวแปรระดับ component ทับระดับ app ได้ หน้าจอตั้งค่า
		// ไม่ได้บอกว่าค่าไหนชนะ บรรทัดนี้บอก และเป็นวิธีเดียวที่ตรวจได้จาก
		// ภายนอกโดยไม่ต้องเปิด endpoint ที่คายค่า config ออกมา
		lg.Info("server starting", "addr", addr, "env", cfg.AppEnv, "version", cfg.Version,
			"cors_origins", cfg.CORSOrigins)
		if err := app.Listen(addr, fiber.ListenConfig{
			EnablePrintRoutes:     !cfg.IsProduction(),
			DisableStartupMessage: cfg.IsProduction(),
		}); err != nil {
			lg.Error("listen failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// ให้เวลาคำขอที่ค้างอยู่ทำงานจนจบ โดยเฉพาะการโพสต์เอกสารที่กำลังตัดสต็อก
	// การตัดกลางคันจะทิ้ง transaction ค้างและ lock ค้างไว้ที่ฐานข้อมูล
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return app.ShutdownWithContext(shutdownCtx)
}

func registerOps(app *fiber.App, db *repository.DB, cfg *config.Config) {
	// healthz บอกว่า process ยังทำงานอยู่ ใช้เป็น liveness probe
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// readyz บอกว่าพร้อมรับงานจริง แยกจาก healthz เพื่อไม่ให้ตัวจัดการคอนเทนเนอร์
	// รีสตาร์ต process ทิ้งเพียงเพราะฐานข้อมูลล่มชั่วคราว ซึ่งการรีสตาร์ตไม่ช่วยอะไร
	app.Get("/readyz", func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c, 3*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).
				JSON(fiber.Map{"status": "unavailable", "reason": "database"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})

	app.Get("/version", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"version": cfg.Version, "api": "v2"})
	})
}

// registerLegacyGone ตอบเส้นทางรุ่นก่อนด้วย 410 แทน 404
//
// 410 แปลว่าเคยมีและถูกถอดออกโดยตั้งใจ ซึ่งบอกผู้เรียกว่าให้ไปแก้โค้ด
// ต่างจาก 404 ที่ชวนให้ไปตรวจว่าพิมพ์ URL ผิดหรือเปล่า
func registerLegacyGone(app *fiber.App) {
	app.All("/api/v1/*", func(c fiber.Ctx) error {
		return httpx.Gone("เส้นทางนี้ถูกถอดออกแล้ว กรุณาใช้ /api/v2/*")
	})
}

// shortID สร้าง trace id สั้น ๆ แทนค่าเริ่มต้นของ requestid ที่ยาว 43 ตัวอักษร
//
// id ตัวนี้ใช้จับคู่บรรทัด log กับสิ่งที่ผู้ใช้แจ้งมาเท่านั้น ไม่ได้ใช้เป็นคีย์
// ในฐานข้อมูลและไม่ต้องเดาไม่ได้ 8 ตัวอักษรจึงพอ และทำให้บรรทัด log อ่านง่ายขึ้นมาก
func shortID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		// ไม่ควรเกิด แต่ถ้าเกิดก็ยังต้องมี id ให้ใช้ ห้ามทำให้คำขอล้มเพราะเรื่องนี้
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}

func parseLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
