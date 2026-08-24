// Path: internal/platform/logx/logx.go
package logx

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Zone เป็น offset ตายตัว +7 ไม่ใช่ชื่อโซน
//
// time.LoadLocation("Asia/Bangkok") ต้องมี tzdata ติดมากับ image ด้วย
// ถ้าไม่มีมันจะตกกลับไปใช้เวลาของเครื่องโดยไม่บอกใคร ซึ่งทำให้เวลาใน log
// ของ container ไม่ตรงกับเวลาที่คนอ่านคาดไว้ ประเทศไทยไม่มี DST
// offset ตายตัวจึงถูกต้องตลอดปี
var Zone = time.FixedZone("UTC+7", 7*60*60)

// TimeLayout ใช้ร่วมกันทั้ง access log และ slog เพื่อให้สองอย่างเรียงคู่กันได้
const TimeLayout = "20060102-15:04:05"

// Handler พิมพ์ slog record เป็นบรรทัดเดียวรูปแบบเดียวกับ access log
//
//	20260824-21:01:52 | WARN  | request rejected | method=GET path=/ status=408
//
// JSON handler ของ slog อ่านด้วยตาลำบากเพราะ key ยาวและมี escape เต็มไปหมด
// ระบบนี้มีคนอ่าน log สดตอนแก้ปัญหามากกว่าเอาเข้าเครื่องมือค้นหา
// จึงเลือกให้คนอ่านง่ายไว้ก่อน
type Handler struct {
	out   io.Writer
	mu    *sync.Mutex
	level slog.Leveler
	// attrs และ groups มาจาก With/WithGroup ซึ่ง slog เรียกซ้ำได้หลายชั้น
	attrs  []slog.Attr
	groups []string
}

func NewHandler(out io.Writer, level slog.Leveler) *Handler {
	return &Handler{out: out, mu: &sync.Mutex{}, level: level}
}

func (h *Handler) Enabled(_ context.Context, l slog.Level) bool {
	return l >= h.level.Level()
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	// ผูกชื่อ group ตั้งแต่ตอนนี้ เพราะ WithGroup ที่เรียกทีหลังต้องไม่มีผล
	// ย้อนหลังกับ attr ที่ใส่ไว้ก่อนแล้ว
	prefix := h.prefix()
	qualified := make([]slog.Attr, 0, len(attrs))
	for _, a := range attrs {
		a.Key = prefix + a.Key
		qualified = append(qualified, a)
	}

	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), qualified...)
	return &clone
}

func (h *Handler) prefix() string {
	if len(h.groups) == 0 {
		return ""
	}
	return strings.Join(h.groups, ".") + "."
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	clone.groups = append(append([]string{}, h.groups...), name)
	return &clone
}

func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.Grow(160)

	t := r.Time
	if t.IsZero() {
		t = time.Now()
	}
	b.WriteString(t.In(Zone).Format(TimeLayout))
	b.WriteString(" | ")

	// level ยาวไม่เท่ากัน เติมช่องว่างให้คอลัมน์ถัดไปตรงกันทุกบรรทัด
	b.WriteString(fmt.Sprintf("%-5s", r.Level.String()))
	b.WriteString(" | ")
	b.WriteString(r.Message)

	first := true
	write := func(prefix string, a slog.Attr) {
		if a.Equal(slog.Attr{}) {
			return
		}
		if first {
			b.WriteString(" | ")
			first = false
		} else {
			b.WriteByte(' ')
		}
		writeAttr(&b, prefix, a)
	}

	// attr จาก WithAttrs ถูกเติมชื่อ group ไว้แล้ว เหลือเฉพาะ attr ของ record
	// ที่ต้องเติมตอนนี้
	for _, a := range h.attrs {
		write("", a)
	}
	prefix := h.prefix()
	r.Attrs(func(a slog.Attr) bool {
		write(prefix, a)
		return true
	})
	b.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := io.WriteString(h.out, b.String())
	return err
}

func writeAttr(b *strings.Builder, prefix string, a slog.Attr) {
	v := a.Value.Resolve()

	// group ที่ส่งมาเป็นค่า ต้องกางออกเป็น key ย่อย ไม่งั้นจะพิมพ์เป็น []
	if v.Kind() == slog.KindGroup {
		inner := prefix + a.Key + "."
		if a.Key == "" {
			inner = prefix
		}
		for i, sub := range v.Group() {
			if i > 0 {
				b.WriteByte(' ')
			}
			writeAttr(b, inner, sub)
		}
		return
	}

	b.WriteString(prefix)
	b.WriteString(a.Key)
	b.WriteByte('=')
	writeValue(b, v)
}

func writeValue(b *strings.Builder, v slog.Value) {
	var s string
	switch v.Kind() {
	case slog.KindString:
		s = v.String()
	case slog.KindInt64:
		s = strconv.FormatInt(v.Int64(), 10)
	case slog.KindDuration:
		// ตัดเศษต่ำกว่ามิลลิวินาทีทิ้ง ความละเอียดระดับนาโนไม่ช่วยคนอ่าน
		s = v.Duration().Round(time.Millisecond).String()
	case slog.KindTime:
		s = v.Time().In(Zone).Format(TimeLayout)
	default:
		s = fmt.Sprint(v.Any())
	}

	// ใส่เครื่องหมายคำพูดเฉพาะตอนที่ค่ามีช่องว่าง ไม่งั้นจะแยกไม่ออกว่า
	// ช่องว่างนั้นคั่นค่าหรืออยู่ในค่า — ข้อความภาษาไทยส่วนใหญ่ไม่มีช่องว่าง
	// จึงไม่โดนใส่เครื่องหมายให้รก
	if s == "" || strings.ContainsAny(s, " \t\n\"") {
		s = strconv.Quote(s)
	}
	b.WriteString(s)
}
