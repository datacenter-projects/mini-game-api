#!/usr/bin/env bash
# ตรวจกฎโครงสร้างที่ตกลงกันไว้ใน CLAUDE.md — CI รันทุก PR, พังข้อไหนต้องแก้ก่อน merge
# รันเอง: make check-structure
set -uo pipefail
cd "$(dirname "$0")/.."

fail=0
violation() { echo "❌ $1"; echo "$2" | sed 's/^/     /'; fail=1; }

# grep เฉพาะไฟล์ .go ที่ไม่ใช่ test และข้ามบรรทัด comment
g() { grep -rnE --include='*.go' --exclude='*_test.go' "$@" 2>/dev/null | grep -vE '^[^:]+:[0-9]+:[[:space:]]*//'; }

# 1. อ่าน env ได้ที่ pkg/configs เท่านั้น
out=$(g 'os\.(Getenv|LookupEnv)\(' app pkg platform | grep -v '^pkg/configs/')
[ -n "$out" ] && violation "os.Getenv นอก pkg/configs — ให้เพิ่ม field ใน configs.Config แทน" "$out"

# 2. repository ห้ามอ้าง global DB และห้ามเปิด transaction
out=$(g 'database\.DBConn' app/repository)
[ -n "$out" ] && violation "repository อ้าง database.DBConn — รับ db *gorm.DB เป็น param แรกแทน" "$out"
out=$(g '\.Transaction\(' app/repository)
[ -n "$out" ] && violation "repository เปิด Transaction — ย้ายไปเปิดที่ service" "$out"

# 3. models ห้ามแตะ DB/redis/service
out=$(g '"app/(platform|app/service|app/repository|app/core)' app/models)
[ -n "$out" ] && violation "models import layer อื่น — models มีได้แค่ struct" "$out"

# 4. core ต้อง pure
out=$(g '"(app/platform|app/pkg/configs|app/app/service|app/app/repository|app/app/internals|app/app/externals|github.com/gofiber|gorm.io|github.com/redis)' app/core)
[ -n "$out" ] && violation "app/core import ของที่มี side effect — core ต้องเป็น pure function" "$out"

# 5. service ห้ามรู้จัก HTTP layer
out=$(g '"(app/app/internals/[a-z_]+/controllers|github.com/gofiber)' app/service)
[ -n "$out" ] && violation "service import fiber/controller — service ต้องไม่รู้จัก HTTP" "$out"

# 6. ห้ามแยก error ด้วยข้อความ
out=$(g 'strings\.(Contains|HasPrefix|HasSuffix|EqualFold)\([^)]*\.Error\(\)' app pkg)
[ -n "$out" ] && violation "แยก error ด้วยข้อความ — ใช้ errors.Is(err, apperr.ErrX)" "$out"

# 7. ห้าม AutoMigrate (schema มาจาก goose เท่านั้น)
out=$(g 'AutoMigrate\(' app pkg platform scripts)
[ -n "$out" ] && violation "AutoMigrate — สร้าง migration ด้วย make migration name=..." "$out"

# 8. ห้าม fmt.Print* ในโค้ดแอป (ใช้ logger)
out=$(g 'fmt\.Print(ln|f)?\(' app pkg platform)
[ -n "$out" ] && violation "fmt.Print — ใช้ logger.Ctx(ctx) แทน" "$out"

# 9. ชื่อ module ต้องตรงกันทุก layer: controllers/{x} ต้องมี dto/{x} และ app/service/{x}
for ctl in app/internals/*/controllers/*/ app/externals/*/controllers/*/; do
  [ -d "$ctl" ] || continue
  mod=$(basename "$ctl")
  ctxdir=$(dirname "$(dirname "$ctl")")
  [ -d "$ctxdir/dto/$mod" ] || violation "ไม่มี dto คู่กับ controller" "$ctl → ต้องมี $ctxdir/dto/$mod/"
  [ -d "app/service/$mod" ] || violation "ไม่มี service คู่กับ controller" "$ctl → ต้องมี app/service/$mod/"
done

# 10. ทุก migration ต้องมี Up และ Down
for f in database/migrations/*.sql; do
  grep -q -- '-- +goose Up' "$f" && grep -q -- '-- +goose Down' "$f" || violation "migration ไม่ครบ Up/Down" "$f"
done

if [ $fail -eq 0 ]; then echo "✅ structure OK"; fi
exit $fail
