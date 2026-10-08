# mini-game-api

Backend API ของระบบ mini-game (Go / Fiber / PostgreSQL / Redis)

## เริ่มต้น

```bash
cp .env.example .env
make dev-up      # postgres + redis ผ่าน docker compose
make run         # http://localhost:8181/health/ready
```

## คำสั่ง

| คำสั่ง | ทำอะไร |
|---|---|
| `make run` | รัน server |
| `make check` | ตรวจโครงสร้าง + build + vet + unit test (รันก่อนเปิด PR) |
| `make test` / `make test-integration` | unit / integration test |
| `make migration name=agent_create_tables` | สร้างไฟล์ migration ใหม่ |
| `make migrate-up` / `migrate-down` / `migrate-status` | จัดการ migration |

## เอกสาร

- [CLAUDE.md](CLAUDE.md) — กฎบังคับของโปรเจกต์
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) — โครงสร้างและแม่แบบโค้ด **อ่านก่อนเขียนโค้ด**
- [docs/MIGRATION_STATUS.md](docs/MIGRATION_STATUS.md) — สถานะการย้ายจากระบบเก่า
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) — ออก tag เพื่อ build image และสิ่งที่ k8s ต้องเตรียม
