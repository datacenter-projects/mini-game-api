-- agent_management MGMT-19 (lead review 2026-10-09 H2 / H3 / Q-R1): Share Master ใช้ค่าตาม Company Seamless Master (CSM) อัตโนมัติ
-- ปรับ Share Master ที่มีอยู่ให้ตรงกับ CSM ของตัวเองครั้งเดียว: pt_from_parent = ของ CSM · force / remain = 0 · status · status_game ตาม CSM
-- commission ไม่แตะ (CSM ตั้งแยกต่อ Share Master) · หลังจากนี้ service ปรับให้เองทุกครั้งที่ Superadmin แก้ CSM

-- +goose Up
UPDATE agent_game_settings sm
SET pt_from_parent = csm.pt_from_parent, force = 0, remain = 0, status = csm.status, status_game = csm.status_game, updated_at = now()
FROM user_agents a, agent_game_settings csm, user_agents p
WHERE a.id = sm.agent_id AND a.role = 'SHAREHOLDER' AND a.agent_type = 'MASTER'
  AND p.id = a.parent_id AND p.role = 'COMPANY' AND p.agent_type = 'SEAMLESS_MASTER'
  AND csm.agent_id = p.id AND csm.game_code = sm.game_code
  AND (sm.pt_from_parent <> csm.pt_from_parent OR sm.force <> 0 OR sm.remain <> 0 OR sm.status <> csm.status OR sm.status_game <> csm.status_game);

-- R2: remain_quota ของ Member ใต้ Share Master = ค่าที่ Share Master ได้รับ − pt (กรณี pt_from_parent ถูกปรับข้างบน)
UPDATE user_member_game_settings ums
SET remain = round((sm.pt_from_parent - ums.pt)::numeric, 4)::double precision
FROM user_members m, user_agents a, agent_game_settings sm
WHERE m.id = ums.user_member_id AND a.id = m.agent_id AND a.role = 'SHAREHOLDER' AND a.agent_type = 'MASTER'
  AND sm.agent_id = a.id AND sm.game_code = ums.game_code
  AND sm.pt_from_parent >= ums.pt AND ums.remain <> round((sm.pt_from_parent - ums.pt)::numeric, 4)::double precision;

-- +goose Down
-- ค่าเดิมก่อนปรับไม่ได้เก็บไว้ · ไม่คืน
SELECT 1;
