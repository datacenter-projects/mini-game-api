# บัญชี mock บน dev

สร้างด้วย `scripts/seed_mock_users -root superadmin1` (วิธีรันดู [TESTING.md](TESTING.md)) · server `https://dev-mini-api.pirate168.com`

- รหัสผ่าน: ทุกบัญชีใช้รหัสเดียวกัน ขอจากคนที่รัน script (ไม่เขียนไว้ในเอกสาร)
- login ครั้งแรกต้องตั้ง passcode
- ค่าหุ้นส่วน `pt_from_parent` ลดชั้นละ 10 · ใต้ Seamless Master ได้ 90 ตามกฎ · force / remain / commission = 0
- ยอดเงิน: ฝั่ง Transfer ทุกบัญชีมี THB 10,000.00 · ฝั่ง Seamless ไม่มียอดเงิน
- สกุล: บัญชีที่ต้องเลือกสกุลได้ THB

## สาย

```
superadmin1
├─ mockcomtransfer (COMPANY_TRANSFER)
│  ├─ mockshareb2b (SHARE_B2B) → mockagentb2b (AGENT) → mockagentb2b2 (AGENT)
│  └─ mockshareb2c (SHARE_B2C)
│     ├─ mockagentb2c (AGENT) → mockmemagentb2c (MEMBER)
│     └─ mockmemshareb2c (MEMBER)
├─ mockcom1to1 (COMPANY_SEAMLESS_1TO1) → mockmem1to1 (MEMBER)
├─ mockcomreseller (COMPANY_SEAMLESS_RESELLER) → mocksharereseller (SHARE_RESELLER) → mockmemreseller (MEMBER)
└─ mockcommaster (COMPANY_SEAMLESS_MASTER) → mocksharemaster (SHARE_MASTER) → mockmemmaster (MEMBER)
```

## รายชื่อ

| user_type | username | ผู้สร้าง | pt_from_parent | ยอดเงิน THB | sub |
|---|---|---|---|---|---|
| COMPANY_TRANSFER | mockcomtransfer | superadmin1 | 90 | 10,000.00 | mockcomtransfer@staff |
| SHARE_B2B | mockshareb2b | mockcomtransfer | 80 | 10,000.00 | mockshareb2b@staff |
| AGENT | mockagentb2b | mockshareb2b | 70 | 10,000.00 | mockagentb2b@staff |
| AGENT | mockagentb2b2 | mockagentb2b | 60 | 10,000.00 | mockagentb2b2@staff |
| SHARE_B2C | mockshareb2c | mockcomtransfer | 80 | 10,000.00 | mockshareb2c@staff |
| AGENT | mockagentb2c | mockshareb2c | 70 | 10,000.00 | mockagentb2c@staff |
| MEMBER | mockmemagentb2c | mockagentb2c | — | 10,000.00 | — |
| MEMBER | mockmemshareb2c | mockshareb2c | — | 10,000.00 | — |
| COMPANY_SEAMLESS_1TO1 | mockcom1to1 | superadmin1 | 90 | — | mockcom1to1@staff |
| MEMBER | mockmem1to1 | mockcom1to1 | — | — | — |
| COMPANY_SEAMLESS_RESELLER | mockcomreseller | superadmin1 | 90 | — | mockcomreseller@staff |
| SHARE_RESELLER | mocksharereseller | mockcomreseller | 80 | — | mocksharereseller@staff |
| MEMBER | mockmemreseller | mocksharereseller | — | — | — |
| COMPANY_SEAMLESS_MASTER | mockcommaster | superadmin1 | 90 | — | mockcommaster@staff |
| SHARE_MASTER | mocksharemaster | mockcommaster | 90 | — | mocksharemaster@staff |
| MEMBER | mockmemmaster | mocksharemaster | — | — | — |

sub (`{username}@staff`) ได้สิทธิ์ `view` ทุกเมนูของเจ้าของ · Member ไม่มี sub

หมายเหตุ (2026-10-09): Share Reseller / Share Master สร้างได้แค่ Member แล้ว (lead H1) · บน dev ยังมี `mockagentreseller` และ `mockagentmaster` ที่สร้างไว้ก่อนกฎนี้ (Member ของสองสายนี้อยู่ใต้ Agent ตัวนั้น) · รัน script ใหม่ด้วย `-prefix` อื่นจะได้สายตามตารางนี้
