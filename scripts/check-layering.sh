#!/usr/bin/env sh
set -eu

# ใช้ grep -E ของ POSIX ไม่ใช่ ripgrep เพราะ runner ของ CI ไม่ได้ติดตั้ง rg มาให้
# แล้ว guard ที่เคยเช็ค rg ก็ทำให้ทุก step หลังจากนี้ไม่ได้รันเลยตั้งแต่ commit ไหนก็ไม่รู้
#
# grep คืน exit 1 เมื่อไม่เจอ ซึ่งกับ `set -e` จะทำให้สคริปต์ตายทั้งที่เป็นผลลัพธ์ที่เราต้องการ
# จึงต้องเรียกใต้ `if` เสมอ เพราะ `set -e` ไม่สนใจ exit code ของเงื่อนไข
#
# ทุกการค้นจำกัดไว้ที่ไฟล์ .go เท่านั้น เพื่อไม่ให้ไฟล์อย่าง go.sum หรือ testdata
# ทำให้ผลลัพธ์เพี้ยน

# ชั้นข้อมูลและการเข้าถึงฐานข้อมูลต้องใช้งานได้โดยไม่ต้องมี HTTP server
for dir in internal/repository internal/resources internal/schema; do
  if grep -rnE --include='*.go' 'github\.com/gofiber/fiber' "$dir"; then
    echo "layering violation: $dir must not import Fiber" >&2
    exit 1
  fi
done

# package schema คืน error ระดับแอปได้ แต่ต้องไม่รู้จัก domain หรือ crud ที่เป็นชั้นบน
if grep -rnE --include='*.go' '"penbun/api/internal/(crud|domain|repository|resources)' internal/schema; then
  echo 'layering violation: schema may not depend on higher layers' >&2
  exit 1
fi
