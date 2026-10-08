# เอกสาร API สำหรับหน้าบ้าน

สรุปจาก spec ที่อนุมัติแล้วใน `docs/modules/` และโค้ดจริง · ใช้ส่งทีมหน้าบ้าน · มีตัวอย่าง curl ทุกเส้น

| ไฟล์ | เนื้อหา | spec ต้นทาง |
|---|---|---|
| [auth.md](auth.md) | login, logout, passcode, รหัสผ่าน, admin reset, flow หลัง login, บัญชีถูกระงับ, error codes | `agent_auth.md`, `agent_auth_phase2.md` |
| [account.md](account.md) | Profile, ข้อมูลรับรอง API, Dashboard | `account.md` |
| [agent_management.md](agent_management.md) | สร้างบัญชี, รายชื่อ / ค้นหาดาวน์ไลน์, รายละเอียด, แก้ข้อมูล / สถานะ / PT, บัญชีย่อย | `agent_management.md` |

แก้ spec แล้วต้องแก้ไฟล์ในโฟลเดอร์นี้ใน PR เดียวกัน
