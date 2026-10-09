package apperr

// agent_management (module id 02) — spec: docs/modules/agent_management.md หัวข้อ 9
var (
	ErrCannotCreateType  = New(402301, "สร้างบัญชีประเภทนี้ไม่ได้", "You cannot create this account type")
	ErrNoMenuPermission  = New(402303, "ไม่มีสิทธิ์ใช้งานเมนูนี้", "You do not have permission for this menu")
	ErrNotDirectCreator  = New(402304, "แก้ไขได้เฉพาะผู้สร้างของบัญชีนี้", "Only the direct creator can change this account")
	ErrPTExceedsReceived = New(402305, "ค่าที่ให้ลูกเกินกว่าที่ได้รับ", "PT given to the child exceeds what you received")
	// ErrPTBelowChildUsage ใช้คู่กับ WithMessage เพื่อบอกค่าต่ำสุดที่ตั้งได้ (MGMT-24)
	ErrPTBelowChildUsage      = New(402306, "ค่าที่ให้ลูกต่ำกว่าที่ลูกใช้อยู่", "PT given is lower than what the child already uses")
	ErrSeamlessMasterPTLocked = New(402307, "Share Master ใช้ค่า PT ตาม Company Seamless Master", "Share Master uses the PT of its Company Seamless Master")
	ErrForceRemainExceeded    = New(402308, "Force / Remain เกินที่กำหนด", "Force / Remain exceeds the allowed value")
	ErrCommissionExceeded     = New(402309, "Commission เกินที่กำหนด", "Commission exceeds the allowed value")
	ErrCurrencyNotInCreator   = New(402310, "สกุลเงินไม่อยู่ในสกุลของผู้สร้าง", "Currency is not one of the creator's currencies")
	ErrSubaccountNotAllowed   = New(402311, "บัญชีย่อยทำรายการนี้ไม่ได้", "Subaccounts cannot do this")
	ErrInsufficientInitial    = New(402312, "ยอดเงินของคุณไม่พอสำหรับยอดเงินตั้งต้น", "Your balance is not enough for the initial balance")

	ErrUsernameTaken      = New(402401, "username นี้ถูกใช้แล้ว", "Username is already taken")
	ErrDownlineNotFound   = New(402402, "ไม่พบบัญชีในสายของคุณ", "Account not found in your downline")
	ErrPhoneTaken         = New(402403, "เบอร์โทรนี้ถูกใช้แล้ว", "Phone number is already taken")
	ErrSubaccountNotFound = New(402404, "ไม่พบบัญชีย่อยของคุณ", "Subaccount not found")
)
