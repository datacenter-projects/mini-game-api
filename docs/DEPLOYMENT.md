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

`ci.yml` รันเฉพาะ PR และตอนถูก release เรียก — push เข้า `dev`/`main` (รวม merge ในเครื่องแล้ว push) **ไม่รัน ci** ผลของ check ใน release tag คือการทดสอบของ commit นั้น

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
| port | `8181` (HTTP API), `9090` (Prometheus `/metrics` ภายใน cluster — ดู [Monitoring](#monitoring-prometheus)) |
| user | non-root uid `10001` — ตั้ง `securityContext.runAsNonRoot: true`, `runAsUser: 10001`, `allowPrivilegeEscalation: false` (เปิด `readOnlyRootFilesystem: true` ได้ เพราะ app ไม่เขียนไฟล์) |
| liveness probe | `GET /health/live` :8181 — เช็คแค่ว่า process ตอบได้ |
| readiness probe | `GET /health/ready` :8181 — ping Postgres + Redis ไม่พร้อมได้ 503 |
| root `/` | `GET` / `HEAD /` :8181 ตอบ 200 เสมอ (ไม่เช็ค dependency) ไว้ให้ load balancer / uptime check และคนเปิด domain — **ไม่ใช้เป็น k8s probe** probe ต้องใช้ `/health/live` และ `/health/ready` |
| shutdown | รับ SIGTERM แล้วปิดภายใน `SHUTDOWN_TIMEOUT` (default 30s) — `terminationGracePeriodSeconds` ต้องมากกว่าค่านี้ |
| timezone | image ตั้ง `TZ=Asia/Bangkok` ไว้แล้ว |

`HEALTHCHECK` ใน Dockerfile ใช้กับ docker เท่านั้น k8s ไม่อ่าน ต้องตั้ง probe เอง

### Environment

ชื่อ key ทั้งหมดดูที่ `.env.example` (ต้นฉบับคือ `pkg/configs/config.go`) ค่าที่เป็นความลับเก็บใน k8s Secret

> ⚠️ **`TRUSTED_PROXIES` — dev / uat / prod ต้องตั้งเสมอ:** app อยู่หลัง ingress / LB ถ้าไม่ตั้ง **ทุกคนจะได้ IP ของ ingress** → rate limit ของ BO login และ player login ใช้ร่วมกันทั้งระบบ (คนเดียวรัวก็ล็อกทุกคน) + whitelist ของ operator (`transfer/*`) ไม่ผ่าน (`407305`) · ต้องใส่ CIDR ของ proxy **ทุกชั้น** (ALB / nginx pod / CDN ถ้ามี) · ถ้าไม่ตั้งนอก `APP_ENV=local` app จะ boot ได้แต่ log `warn` ตั้งแต่เริ่ม (`TRUSTED_PROXIES is empty …`) · ตั้งผิดรูปแบบ = boot ไม่ขึ้น

| key | หมายเหตุ |
|---|---|
| `APP_ENV` | `dev` / `prod` |
| `SERVER_ADDR` | `:8181` |
| `METRICS_ADDR` | `:9090` (default) — ค่าว่าง = ปิด metrics ต้องไม่ซ้ำกับ `SERVER_ADDR` |
| `TRUSTED_PROXIES` | CIDR (หรือ IP เดี่ยว) ของ ingress/LB **ทุกชั้นที่อยู่หน้า app** คั่นด้วย `,` — ว่าง = ไม่เชื่อ `X-Forwarded-For` (IP ที่ log / rate limit / whitelist จะเป็น IP ของ proxy) · เมื่อตั้ง ระบบไล่ `X-Forwarded-For` **จากขวาไปซ้าย ข้ามทุก IP ที่อยู่ใน `TRUSTED_PROXIES`** แล้วใช้ IP แรกที่ไม่ใช่ proxy เป็น IP ของ client (`middleware.ClientIP`) — ค่าที่ client ใส่เองจะอยู่ทางซ้ายจึงปลอมไม่ได้ · **รายการที่พิมพ์ผิด (ไม่ใช่ IP / CIDR) = boot ไม่ขึ้น** · ถ้าลืมใส่ proxy ชั้นใดชั้นหนึ่ง IP ของ proxy ชั้นนั้นจะถูกนับเป็น client (ทุกคนใช้ IP เดียวกัน) · connection ที่ไม่ได้มาจาก proxy ใน list ไม่เชื่อ header |
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
| `PLAYER_SESSION_*`, `OPERATOR_IP_LIMIT_PER_MINUTE` | มี default ใช้ตาม `.env.example` ได้ (player_auth) |
| `PLAYER_LOBBY_URL` | URL หน้า lobby ที่ `transfer/login` ต่อ `?token=` ท้าย — **prod / dev ต้องตั้ง** ไม่ตั้ง = `transfer/login` ตอบ 503 · ต้องเป็น absolute `http(s)://host/...` ผิดรูปแบบ = **boot ไม่ขึ้น** |
| `MIGRATE_ON_START` | **`false`** (default เมื่อไม่ตั้ง) — migration รันแยกตามข้อถัดไป |
| `IS_CRONJOB` | **`false`** (default) — `true` = pod นี้รัน cronjob ทุกตัว (`app/cronjob` — ตอนนี้ `currency_rate_sync`) · **ตั้ง `true` เฉพาะ deployment เดียวต่อ env** (pod อื่นไม่รัน) · แทน `IS_CURRENCY_RATE_SYNC` (ลบแล้ว — ตั้งไว้ก็ไม่มีผล) · เปิด / ปิดแต่ละ job ด้วยตาราง `cronjobs` (ดู "งานตามเวลา" ด้านล่าง) |
| `ASKMELOTTO_API_BASE_URL` | origin ของระบบ askmelotto เช่น `https://dev-api.amblotto.net` (เปล่า ไม่มี path / query / `/` ท้าย ไม่งั้น boot ไม่ขึ้น) · **uat / prod ต้องตั้ง** (ไม่ตั้ง = boot ไม่ขึ้น) · local / dev ไม่ตั้ง = host ฝั่ง dev · แทน `CURRENCY_RATE_API_BASE_URL` (ลบแล้ว) · pod ที่ `IS_CRONJOB=true` ต้องออกไปเรียก URL นี้ได้ (HTTPS ออก) |

key ที่ required ถ้าขาด app จะไม่ boot (log `missing required env: ...`)

### Monitoring (Prometheus)

app เปิด `/metrics` บน **พอร์ตแยก `9090`** (`METRICS_ADDR`) ไม่ได้อยู่บนพอร์ต API — พอร์ตนี้ไม่มี auth จึงต้องเปิดแค่ภายใน cluster

- k8s Service เพิ่ม port `9090` (ชื่อเช่น `metrics`) ให้ Prometheus scrape ได้
- **ห้าม** route `9090` ผ่าน Ingress / LoadBalancer — Ingress ชี้ไปพอร์ต `8181` เท่านั้น
- ตั้ง `METRICS_ADDR` เป็นค่าว่าง = ปิด metrics ทั้งหมด (ไม่เปิดพอร์ต ไม่นับ request)

เลือกวิธี scrape อย่างใดอย่างหนึ่ง

ServiceMonitor (Prometheus Operator)

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: minigame-api
spec:
  selector:
    matchLabels:
      app: minigame-api
  endpoints:
    - port: metrics
      path: /metrics
      interval: 30s
```

หรือ annotation บน Pod (Prometheus ที่ใช้ kubernetes_sd + relabel ตาม annotation)

```yaml
metadata:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9090"
    prometheus.io/path: "/metrics"
```

metric ที่มี

| metric | label | หมายเหตุ |
|---|---|---|
| `mini_game_api_http_requests_total` (counter) | `method`, `path`, `status` | ชื่อและ label เดียวกับ v1 — dashboard/alert เดิมใช้ต่อได้ |
| `mini_game_api_http_request_duration_seconds` (histogram) | `method`, `path`, `status` | bucket = Prometheus default |
| `go_*`, `process_*` | — | Go runtime + process (CPU, memory, fd) |
| `go_sql_*` | `db_name` = `write` / `read` | stat ของ Postgres pool (`read` มีเมื่อตั้ง `DB_READ_HOST`) |
| `mini_game_api_redis_pool_{hits,misses,timeouts}_total`, `..._{total,idle}_conns` | — | stat ของ Redis pool |

ข้อควรรู้เรื่อง label ของ HTTP

- `path` เป็น route pattern เช่น `/api/v1/bo/pb/agents/:id` ไม่ใช่ path จริง — path ที่ไม่ตรง route ไหนเป็น `unmatched`, request ที่ middleware ของ group ตอบเอง (เช่น 401) เป็น prefix ของ group
- `/` และ `/health/*` ไม่ถูกนับ (LB poll)
- `status` คือ HTTP status — error ทาง business ตอบ HTTP 200 (ผลอยู่ใน field `code` ของ body) จึงเห็นเป็น 200 ตั้ง alert error rate ด้วย `status=~"5.."`

### งานตามเวลา (cronjob — `currency_rate.md` · CLAUDE.md กฎข้อ 30–31)

- งานตามเวลารันใน process เดียวกับ API เฉพาะ pod ที่ `IS_CRONJOB=true` — **ตั้งเฉพาะ deployment เดียวต่อ env** (ถ้าตั้งหลาย deployment ยังมี Redis lock กันซ้ำ แต่เปลือง) · เปิด pod เพิ่ม / scale ของ deployment อื่นไม่เกี่ยว
- ตาราง `cronjobs` เก็บ `enabled` ของแต่ละ job + ผลรอบล่าสุด (`last_started_at` · `last_finished_at` · `last_status` = `SUCCESS` / `FAILED` / `SKIPPED` · `last_error`) · แถวสร้างเองตอน boot ครั้งแรกตามค่าเริ่มต้นในโค้ด (`currency_rate_sync` = เปิด) และ **ไม่ถูกทับตอน boot ครั้งถัดไป**
- **เปิด / ปิด job ด้วย SQL** (บน env ร่วม dev / uat / prod **ขอ lead ก่อนทุกครั้ง** · ตั้ง `updated_by = 'sql:{ชื่อ}'`):

```sql
-- ปิด currency_rate_sync (ข้ามรอบเงียบ ไม่ยิง askmelotto ไม่แก้ last_*)
UPDATE cronjobs SET enabled = false, updated_by = 'sql:zerph', updated_at = now() WHERE code = 'currency_rate_sync';
-- เปิดกลับ
UPDATE cronjobs SET enabled = true,  updated_by = 'sql:zerph', updated_at = now() WHERE code = 'currency_rate_sync';
-- ดูสถานะ
SELECT code, enabled, last_status, last_started_at, last_finished_at, last_error, updated_by FROM cronjobs;
```

- **ก่อน deploy ครั้งแรกที่มี `currency_rate` ใหม่:** migrate (`currency_rates` + `cronjobs`) · ตั้ง `IS_CRONJOB=true` ที่ deployment เดียว · pod นั้นออกไปเรียก askmelotto ได้ · env เดิม `IS_CURRENCY_RATE_SYNC` / `CURRENCY_RATE_API_BASE_URL` ลบออกได้ (ไม่มีผลแล้ว) · key Redis เดิม `currency_rate:usdt` / `:meta` ปล่อยไว้ ไม่ต้องลบ
- **alert ที่แนะนำ:** `time() - mini_game_api_cronjob_last_success_timestamp{job="currency_rate_sync"} > 3600` (เรทรายเดือนครบแล้ว job ก็ยัง SUCCESS ทุกรอบโดยไม่ยิง — ถ้าเกิน 1 ชม. แปลว่า job ล้ม / pod cronjob หาย) · `cronjob_runs_total{status="FAILED"}` เพิ่มต่อเนื่อง
- **เติม / แก้เรทด้วยมือ** (เช่น วันที่ 1 ของเดือน askmelotto ช้า หรือต้องแก้ค่า) — ใช้ service เดียวกับ job · env ร่วมขอ lead ก่อน:

```bash
go run ./scripts/currency_rate_sync -month 2026-10            # เติมเฉพาะสกุลที่ขาด (ไม่ทับของเดิม)
go run ./scripts/currency_rate_sync -month 2026-10 -force -yes   # เขียนทับทั้งเดือน (-yes บังคับบน env ร่วม)
```

  สคริปต์พิมพ์ `env` / DB host / askmelotto URL ก่อนทำงานทุกครั้ง · `-force` บน env ร่วมต้องมี `-yes`

### Migration

รัน `/app/migrate up` จาก image tag เดียวกับที่จะ deploy **ก่อน** rollout เสมอ เลือกทางใดทางหนึ่ง

- **Job** (แนะนำ) — Job ต่อ release เช่น `minigame-migrate-<tag>` ต้องสำเร็จก่อนสั่ง rollout Deployment ใช้ได้กับ Helm pre-upgrade hook หรือ Argo CD PreSync
- **initContainer** — ใส่ใน Deployment ใช้ image เดียวกันแต่ `command: ["/app/migrate", "up"]` ทุก pod จะรัน (goose ล็อกตาราง version ไว้ แต่ pod ขึ้นช้ากว่า)

ข้อควรรู้

- ไฟล์ migration ถูก embed ใน binary แล้ว ไม่ต้อง mount อะไรเพิ่ม
- `/app/migrate` โหลด config ชุดเดียวกับ app จึงต้องได้ env ครบเหมือน Deployment (รวม `REDIS_ADDR`, `JWT_SECRET`) แม้จะใช้แค่ Postgres
- migrate fail = exit code ≠ 0 → ห้าม rollout ต่อ
- migration ต้องเข้ากันได้กับ pod เวอร์ชันเก่าที่ยังรันอยู่ระหว่าง rolling update (เพิ่มคอลัมน์ก่อน ลบทีหลังในอีก release)

### รัน seed รูปค่ากลางของ asset (GLOBAL — `asset.md` AST-61)

รูปค่ากลางของ asset เก็บใน repo ที่ `seed/global-assets/{game_code}/{asset_type}.{ext}` (ตอนนี้มี `scratch_card` 19 ช่อง) — **ไม่ถูกใส่ใน image และไม่มี Job ตอน deploy**
คนรันสคริปต์เอง**ทีละ env** หลัง migrate แล้วและ S3 / CloudFront พร้อม:

```
make seed-asset-global
# หรือ: go run ./scripts/seed_asset_global -dir seed/global-assets
```

- **เขียน S3 + DB ของจริงตาม `.env` ของเครื่องที่รัน** → ขอ lead ก่อนรันบน dev / uat / prod ทุกครั้ง · ห้ามรันโดยไม่ได้รับอนุญาต
- `.env` ต้องมี `DB_*` ของ env เป้าหมาย · `S3_BUCKET` · `S3_REGION` · `S3_CDN_BASE_URL` · credential AWS ที่เขียน bucket ได้ · ไม่ตั้ง `S3_BUCKET` = สคริปต์หยุดตั้งแต่ต้น ไม่เขียนอะไร
- **รันซ้ำได้:** ช่องที่ GLOBAL มีรูปแล้วถูกข้าม · เติมเฉพาะช่องว่าง · เปลี่ยนรูปที่ใช้อยู่ = ใส่ `-force` (อัปเวอร์ชันใหม่ + ตั้งเป็นรูปที่ใช้ · ไฟล์เดิมยังอยู่ใน bucket / คลัง)
- ท้ายผลบอกช่องของทุกเกมที่ GLOBAL ยังไม่มีรูป (`coin_toss` / `rock_paper_scissors` จะค้างจนกว่าทีมเกมส่งรูป — ไม่ใช่ error) · ต้องครบทุกช่องของ `scratch_card` ก่อน UAT (AST-24)
- ตรวจหลังรัน: `used/list` (`scope: GLOBAL`, `game_code: scratch_card`) ทุกช่องต้องมี `url` และเปิดดูรูปได้

### prod ครั้งแรก (lead 2026-10-11)

prod เริ่มจาก **DB ว่าง** ไม่ย้ายข้อมูลจากระบบเก่า (ถ้าวันหน้าต้องย้ายจะวางแผนเป็นงานแยก)
migration ชุดก่อน prod ที่แปลง / ล้างข้อมูลในไฟล์เดียวกับการเปลี่ยน schema จึง**รัน `migrate up` ตรงได้เลย** ไม่ต้องแยกขั้น เพราะไม่มีแถวให้แปลง:

| migration | ข้อมูลที่แปลง / ล้าง |
|---|---|
| `20261009120000_agent_management_chain` | backfill `user_agents.cnf` · `agent_game_settings.parent_id` |
| `20261009120050_member_management_member_pt` | ตั้ง `pt` / `remain` ของ Member จากค่าของผู้สร้าง |
| `20261009120100_member_management_member_cnf_credit` | `user_members.cnf` · ย้ายยอดไป `credit` · DROP `user_member_balances` |
| `20261009140000_agent_management_drop_own_pt` | DROP คอลัมน์ `agent_game_settings.pt` (ค่าเดิมหาย) |
| `20261009140100_member_management_settings_parent` | backfill `user_member_game_settings.parent_id` |
| `20261009150000_agent_management_drop_pt_permission` | ลบ key `pt` ออกจาก `subaccounts.permissions` |
| `20261009160000_agent_management_phone_country_code` · `20261011120000_member_management_phone_country_code` | แปลงเบอร์เป็น 2 field · ล้างเบอร์ที่แปลงไม่ได้ / ซ้ำ (`migration_phone_cleared`) |
| `20261009170000_agent_management_share_master_follow_csm` | ตั้ง Share Master ตาม CSM · Member ที่คิด remain ไม่ได้บันทึกใน `migration_member_remain_skipped` |

ไม่แก้ไฟล์เหล่านี้ (CLAUDE.md กฎข้อ 20)

### migration ที่แปลง / ล้างข้อมูล — หลัง prod มีข้อมูลจริง (lead 2026-10-11)

**ต้องแยก 3 ขั้น** เมื่อตารางที่แตะมีเกิน **100,000 แถว** หรือคาดว่า lock นานเกิน **5 วินาที**:

1. **migration schema** — เพิ่มคอลัมน์แบบ nullable / ไม่มี default ที่ต้องเขียนทุกแถว · pod เวอร์ชันเก่ายังทำงานได้
2. **script แปลงข้อมูล** (`scripts/…`) — ทีละ batch **10,000 แถว** (เรียงตาม primary key · commit ทีละ batch) · รันซ้ำได้ (ข้ามแถวที่แปลงแล้ว) · บันทึกแถวที่ถูกล้างเหมือน `migration_phone_cleared` · รันก่อนขั้น 3 และจดผลใน PR
3. **migration constraint / index** — `ADD CONSTRAINT … NOT VALID` แล้ว `VALIDATE CONSTRAINT` · index ใช้ `CREATE INDEX CONCURRENTLY` (ไฟล์นี้ใส่ `-- +goose NO TRANSACTION` และมี statement เดียวต่อ index)

ตารางเล็กกว่าเกณฑ์ทำในไฟล์เดียวได้เหมือนเดิม · ระบุใน PR ว่าตารางมีกี่แถว

### Rollback

deploy image tag ก่อนหน้ากลับไป ไม่ต้องรัน `migrate down` ถ้า migration เข้ากันได้ย้อนหลังตามข้อบน ถ้าต้อง down จริงให้ lead ตัดสินใจเป็นกรณีไป
