# สถานะงานสำหรับ lead

1 ไฟล์ต่อ branch · ไฟล์อยู่ใน branch นั้นเอง และอัปเดตพร้อม commit งานใน branch นั้น
ชื่อไฟล์ไม่ซ้ำกัน ตอน merge จึงไม่ conflict — `dev` มีสถานะของทุก branch ที่ merge แล้ว

| ไฟล์ | branch |
|---|---|
| `bo-account.md` | `boiledegg/bo/account` |
| `bo-management.md` | `boiledegg/bo/management` |
| `bo-currency-rate.md` | `boiledegg/bo/currency-rate` |
| `test-infra.md` | `boiledegg/test-infra` |

## โครงของแต่ละไฟล์

1. branch · commit ล่าสุด · สถานะ spec
2. เสร็จแล้ว / กำลังทำ / ติดอะไร
3. คำถามที่รอ lead
4. ประวัติอัปเดต — ต่อท้ายทุกครั้งพร้อมวันที่ (ไม่ลบของเก่า)

## วิธีอัปเดต

- แก้หัวข้อ 1–3 ให้ตรงปัจจุบัน และเพิ่ม 1 บรรทัดในประวัติอัปเดต
- commit ไฟล์นี้ไปพร้อม commit งาน หรือแยก commit `docs(lead): ...`
- branch ใหม่: เพิ่มไฟล์ของ branch นั้น และเพิ่มแถวในตารางด้านบน
