// Path: internal/platform/mw/logger.go
package mw

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"

	"penbun/api/internal/platform/logx"
)

const apiPrefix = "/api/v2"

// AccessLog พิมพ์บรรทัดเดียวต่อหนึ่งคำขอในรูปแบบ
//
//	20260824-19:24:35 | POST /auth/login | 401 | ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง
func AccessLog() fiber.Handler {
	return logger.New(logger.Config{
		// สีทำให้มี escape code ปนในบรรทัดและ ${status} ถูก pad เป็น %3d
		// ปิดไปเพื่อให้รูปแบบตรงตามที่ตกลงไว้และ grep ได้ตรง ๆ
		DisableColors: true,
		Format:        "${localtime} | ${method} ${shortpath} | ${status} | ${resmessage}\n",
		CustomTags: map[string]logger.LogFunc{
			// ไม่ใช้ ${time} เพราะค่านั้นมาจาก cache ที่อัปเดตทุก TimeInterval
			// (ค่าเริ่มต้น 500ms) และผูกกับ TimeZone ที่เป็นสตริง
			"localtime": func(out logger.Buffer, _ fiber.Ctx, _ *logger.Data, _ string) (int, error) {
				return out.WriteString(time.Now().In(logx.Zone).Format(logx.TimeLayout))
			},
			"shortpath": func(out logger.Buffer, c fiber.Ctx, _ *logger.Data, _ string) (int, error) {
				return out.WriteString(shortPath(c.Path()))
			},
			"resmessage": func(out logger.Buffer, c fiber.Ctx, _ *logger.Data, _ string) (int, error) {
				return out.WriteString(responseMessage(c))
			},
		},
	})
}

// shortPath ตัด prefix ของ API ออกเพื่อให้เหลือเฉพาะส่วนที่ต่างกันจริง
func shortPath(path string) string {
	trimmed := strings.TrimPrefix(path, apiPrefix)
	if trimmed == "" {
		return "/"
	}
	return trimmed
}

// responseMessage ดึงค่า message จาก JSON ที่ตอบกลับไป
//
// logger ทำงานหลัง handler จบแล้ว body จึงถูกเขียนครบและอ่านได้ตรงนี้
// คำตอบที่สำเร็จมักไม่มี message เลยเพราะข้อความอธิบายไม่มีประโยชน์กับ client
// กรณีนั้นเติมข้อความมาตรฐานของ status code แทน เพื่อไม่ให้ช่องท้ายบรรทัดว่าง
func responseMessage(c fiber.Ctx) string {
	status := c.Response().StatusCode()

	if bytes.Contains(c.Response().Header.ContentType(), []byte("application/json")) {
		if msg := jsonMessage(c.Response().Body()); msg != "" {
			return msg
		}
	}
	return http.StatusText(status)
}

// jsonMessage อ่าน body แบบ stream แล้วหยุดทันทีที่เจอคีย์ message
//
// ไม่ใช้ json.Unmarshal ทั้งก้อนเพราะ response ของ endpoint แบบ list มี data
// เป็นอาเรย์ยาว การ parse ทิ้งทุกคำขอเพื่อเอาข้อความเดียวไม่คุ้ม
// Envelope วาง message ไว้ก่อน data อยู่แล้ว ตัว decoder จึงจบงานก่อนถึงของหนัก
func jsonMessage(body []byte) string {
	dec := json.NewDecoder(bytes.NewReader(body))

	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return ""
	}

	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return ""
		}
		if name, _ := key.(string); name == "message" {
			var msg string
			if err := dec.Decode(&msg); err != nil {
				return ""
			}
			return msg
		}
		// ต้องกินค่าของคีย์ที่ไม่สนใจทิ้ง ไม่งั้น token ถัดไปจะไม่ใช่ชื่อคีย์
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return ""
		}
	}
	return ""
}
