# Deployment

repo นี้สร้างแค่ image แล้ว push ขึ้น GHCR ส่วน k8s manifest (Deployment, Job, Secret, Ingress) อยู่ฝั่ง infra นอก repo
เอกสารนี้บอกว่าฝั่ง infra ต้องเตรียมอะไร

## 1. สร้าง image (`.github/workflows/release.yml`)

push tag แล้ว GitHub Actions จะทำตามลำดับนี้

1. `verify-tag` — tag ต้องอยู่บน branch ที่ถูกต้อง (เช็คด้วย `git merge-base --is-ancestor`)

   | tag | ต้องอยู่บน |
   |---|---|
   | `dev-<เลข>` | `origin/dev` |
   | `prod-<เลข>` | `origin/main` |

2. `check` — เรียก `.github/workflows/ci.yml` ตัวเดียวกับ PR (structure + lint + build + migrate up/down/up + test กับ postgres/redis)
3. `image` — build `Dockerfile` แล้ว push ทำเฉพาะเมื่อสองขั้นแรกผ่าน

image ที่ได้

```
ghcr.io/datacenter-projects/minigame-api:<tag>         # เช่น dev-202610071530
ghcr.io/datacenter-projects/minigame-api:sha-<7 ตัว>   # commit เดียวกันได้ image เดียวกัน
```

label: `org.opencontainers.image.source` (ลิงก์ repo), `org.opencontainers.image.revision` (commit SHA), `org.opencontainers.image.version` (tag)

### วิธีออก tag

```bash
make release env=dev    # tag dev-<YYYYMMDDHHmm> ที่ปลาย origin/dev
make release env=prod   # tag prod-<YYYYMMDDHHmm> ที่ปลาย origin/main (ต้องพิมพ์ yes ยืนยัน)
```

`scripts/release_tag.sh` ทำให้
- `git fetch` แล้ว tag ที่ปลาย `origin/<branch>` เสมอ ไม่ใช้ HEAD ในเครื่อง (อยู่ branch ไหนก็ได้ มี commit ค้างก็ไม่ติดไปด้วย)
- ตั้งชื่อด้วยเวลาไทย และ fail ถ้า branch นั้นยังไม่มี `release.yml`
- push ไม่ผ่าน (เช่นชื่อซ้ำในนาทีเดียวกัน) จะลบ tag ในเครื่องทิ้ง

ทำมือก็ได้ แต่ต้อง tag ที่ `origin/<branch>` ไม่ใช่ HEAD

```bash
git fetch origin
git tag dev-202610071530 origin/dev
git push origin dev-202610071530
```

tag ที่ไม่ได้อยู่บน branch ที่กำหนด workflow จะ fail และไม่ push image

### ครั้งแรกที่ push image

- login GHCR ใช้ `GITHUB_TOKEN` + `permissions: packages: write` ไม่ใช้ PAT
- package `minigame-api` จะผูกกับ repo จาก label `org.opencontainers.image.source` ถ้ามี package ชื่อนี้อยู่แล้วแต่ไม่ได้ผูก repo ให้ org admin เพิ่ม repo `mini-game-api` เป็น **Write** ใน Package settings → Manage Actions access
- package ใหม่เป็น private ฝั่ง k8s ต้องมี `imagePullSecret` (PAT หรือ token ที่มี `read:packages`)

## 2. สิ่งที่ k8s ต้องเตรียม

### Container

| หัวข้อ | ค่า |
|---|---|
| image | `ghcr.io/datacenter-projects/minigame-api:<tag>` |
| entrypoint | `/app/apiserver` (ค่า default ของ image) |
| port | `8181` (HTTP) |
| user | non-root uid `10001` — ตั้ง `securityContext.runAsNonRoot: true`, `runAsUser: 10001`, `allowPrivilegeEscalation: false` (เปิด `readOnlyRootFilesystem: true` ได้ เพราะ app ไม่เขียนไฟล์) |
| liveness probe | `GET /health/live` :8181 — เช็คแค่ว่า process ตอบได้ |
| readiness probe | `GET /health/ready` :8181 — ping Postgres + Redis ไม่พร้อมได้ 503 |
| shutdown | รับ SIGTERM แล้วปิดภายใน `SHUTDOWN_TIMEOUT` (default 30s) — `terminationGracePeriodSeconds` ต้องมากกว่าค่านี้ |
| timezone | image ตั้ง `TZ=Asia/Bangkok` ไว้แล้ว |

`HEALTHCHECK` ใน Dockerfile ใช้กับ docker เท่านั้น k8s ไม่อ่าน ต้องตั้ง probe เอง

### Environment

ชื่อ key ทั้งหมดดูที่ `.env.example` (ต้นฉบับคือ `pkg/configs/config.go`) ค่าที่เป็นความลับเก็บใน k8s Secret

| key | หมายเหตุ |
|---|---|
| `APP_ENV` | `dev` / `prod` |
| `SERVER_ADDR` | `:8181` |
| `TRUSTED_PROXIES` | CIDR ของ ingress/LB คั่นด้วย `,` — ว่าง = ไม่เชื่อ `X-Forwarded-For` (IP ที่ log และ rate limit จะเป็น IP ของ proxy) |
| `CORS_ALLOW_ORIGINS` | domain ของหน้าเว็บ ห้ามใช้ `*` บน prod |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_NAME` | required |
| `DB_PASSWORD` | Secret |
| `DB_SSLMODE` | `require` (default เมื่อไม่ตั้ง) |
| `DB_READ_HOST` | ว่าง = ไม่ใช้ read replica |
| `DB_MAX_OPEN_CONNS` ฯลฯ | pool ต่อ pod — คูณจำนวน replica แล้วต้องไม่เกิน `max_connections` ของ DB |
| `REDIS_ADDR` | required |
| `REDIS_PASSWORD` | Secret |
| `JWT_SECRET` | Secret, required, อย่างน้อย 32 ตัวอักษร, แยกค่าต่อ env |
| `PASSWORD_COST` | bcrypt cost — **12** (default) ห้ามลดบน dev/prod ค่า 4 ใช้กับ CI/test เท่านั้น |
| `SESSION_*`, `LOGIN_*`, `PASSCODE_*`, `TEMP_CREDENTIAL_TTL` | มี default ใช้ตาม `.env.example` ได้ |
| `MIGRATE_ON_START` | **`false`** (default เมื่อไม่ตั้ง) — migration รันแยกตามข้อถัดไป |

key ที่ required ถ้าขาด app จะไม่ boot (log `missing required env: ...`)

### Migration

รัน `/app/migrate up` จาก image tag เดียวกับที่จะ deploy **ก่อน** rollout เสมอ เลือกทางใดทางหนึ่ง

- **Job** (แนะนำ) — Job ต่อ release เช่น `minigame-migrate-<tag>` ต้องสำเร็จก่อนสั่ง rollout Deployment ใช้ได้กับ Helm pre-upgrade hook หรือ Argo CD PreSync
- **initContainer** — ใส่ใน Deployment ใช้ image เดียวกันแต่ `command: ["/app/migrate", "up"]` ทุก pod จะรัน (goose ล็อกตาราง version ไว้ แต่ pod ขึ้นช้ากว่า)

ข้อควรรู้

- ไฟล์ migration ถูก embed ใน binary แล้ว ไม่ต้อง mount อะไรเพิ่ม
- `/app/migrate` โหลด config ชุดเดียวกับ app จึงต้องได้ env ครบเหมือน Deployment (รวม `REDIS_ADDR`, `JWT_SECRET`) แม้จะใช้แค่ Postgres
- migrate fail = exit code ≠ 0 → ห้าม rollout ต่อ
- migration ต้องเข้ากันได้กับ pod เวอร์ชันเก่าที่ยังรันอยู่ระหว่าง rolling update (เพิ่มคอลัมน์ก่อน ลบทีหลังในอีก release)

### Rollback

deploy image tag ก่อนหน้ากลับไป ไม่ต้องรัน `migrate down` ถ้า migration เข้ากันได้ย้อนหลังตามข้อบน ถ้าต้อง down จริงให้ lead ตัดสินใจเป็นกรณีไป
