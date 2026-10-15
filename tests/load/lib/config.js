// ค่าร่วมของ load test — override ได้ด้วย env ของ k6: k6 run -e BASE_URL=http://host:8282 ...
// บัญชีทดสอบสร้างด้วย scripts/seed_loadtest (ค่าเริ่มต้นตรงกับ script นั้น)

export const BASE_URL = __ENV.BASE_URL || 'http://localhost:8282';
export const ACCOUNT_PREFIX = __ENV.ACCOUNT_PREFIX || 'load';
export const ACCOUNT_COUNT = parseInt(__ENV.ACCOUNT_COUNT || '100', 10);
export const PASSWORD = __ENV.PASSWORD || 'LoadTest1!';

export const JSON_HEADERS = { 'Content-Type': 'application/json' };

// accountFor คืน username ของ VU นี้ — VU ละบัญชี (1 บัญชีมีได้ 1 session — AUTH-06)
// ถ้า VU มากกว่าจำนวนบัญชี บัญชีจะถูกใช้ซ้ำและ session เตะกันเอง ให้ seed บัญชีเพิ่ม
export function accountFor(vu) {
  const n = ((vu - 1) % ACCOUNT_COUNT) + 1;
  return `${ACCOUNT_PREFIX}${String(n).padStart(4, '0')}`;
}
