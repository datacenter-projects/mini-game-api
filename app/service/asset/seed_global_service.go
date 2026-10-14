package asset

// ตรรกะของสคริปต์อัปรูปค่ากลาง (GLOBAL) — scripts/seed_asset_global · docs/modules/asset.md AST-61
//
// อ่านไฟล์ที่ทีมเกม / ออกแบบส่งมาตามโครง {dir}/{game_code}/{asset_type}.{png|jpg|jpeg|gif|webp} แล้วอัปเป็น config GLOBAL
// (use_now = true) ผ่าน UploadFileService ตัวเดียวกับ API จริง จึงตรวจกฎ AST-14–17 ครบเหมือนกันทุกข้อ
// ไว้ใน service (ไม่ใช่ใน scripts/) เพื่อให้ integration test อยู่ package เดียวกับ test อื่น — ไม่ล้างตารางชนกัน

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	assetCore "app/app/core/asset"
	assetDto "app/app/internals/backoffice/dto/asset"
	agentAuthService "app/app/service/agent_auth"
	"app/pkg/apperr"
)

// SeedResult — ผลของการรันหนึ่งครั้ง
type SeedResult struct {
	Uploaded int
	Skipped  int
	Failed   int
	Gaps     []GlobalGap // ช่องที่ GLOBAL ยังไม่มีรูปหลังรันจบ
}

var seedImageExts = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true}

// seedMaxCreatedByLen — ความยาวของ created_by ในตาราง (VARCHAR(71))
const seedMaxCreatedByLen = 71

// SeedCreatedBy — ค่า created_by ของไฟล์ที่สคริปต์อัป: script:{user ของเครื่องที่รัน} (แบบ reset_credentials — AUTH-51)
func SeedCreatedBy(osUser string) string {
	s := "script:" + osUser
	if r := []rune(s); len(r) > seedMaxCreatedByLen {
		s = string(r[:seedMaxCreatedByLen])
	}
	return s
}

// SeedGlobalService อัปรูปจาก dir · force = false ข้ามช่องที่ GLOBAL มี used อยู่แล้ว · force = true อัปเวอร์ชันใหม่ + ตั้ง used ทับ (ไฟล์เดิมยังอยู่ในคลัง)
// ไฟล์ที่ผิดทีละไฟล์ไม่หยุดทั้งชุด · รายงานบรรทัดละไฟล์ลง out · ต้องมี storage พร้อมก่อนเรียก (ผู้เรียกตรวจ)
func SeedGlobalService(ctx context.Context, dir string, force bool, by string, out io.Writer) (SeedResult, error) {
	var res SeedResult
	games, err := os.ReadDir(dir)
	if err != nil {
		return res, fmt.Errorf("อ่านโฟลเดอร์ %s ไม่ได้: %w", dir, err)
	}
	actor := agentAuthService.Actor{Username: by}
	target := assetCore.Target{Resolved: true, Scope: assetCore.ScopeGlobal}

	sort.Slice(games, func(i, j int) bool { return games[i].Name() < games[j].Name() })
	for _, g := range games {
		if !g.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(dir, g.Name()))
		if err != nil { // โฟลเดอร์เดียวอ่านไม่ได้ไม่หยุดทั้งชุด — รายงานแล้วไปเกมถัดไป
			res.Failed++
			_, _ = fmt.Fprintf(out, "ล้มเหลว  %s/ — อ่านโฟลเดอร์ไม่ได้: %v\n", g.Name(), err)
			continue
		}
		sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
		perSlot := seedFilesPerSlot(files)
		for _, f := range files {
			ext := strings.ToLower(filepath.Ext(f.Name()))
			if f.IsDir() || !seedImageExts[ext] {
				continue
			}
			slot := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
			label := g.Name() + "/" + slot
			path := filepath.Join(dir, g.Name(), f.Name())

			if names := perSlot[slot]; len(names) > 1 { // ไม่เดาว่าไฟล์ไหนถูก — ให้ทีมเหลือไฟล์เดียวแล้วรันใหม่
				res.Failed++
				_, _ = fmt.Fprintf(out, "ล้มเหลว  %s — มีหลายไฟล์ของช่องเดียวกัน (%s) ให้เหลือไฟล์เดียว\n", label+filepath.Ext(f.Name()), strings.Join(names, ", "))
				continue
			}

			if !force {
				has, err := HasGlobalUsedService(ctx, g.Name(), slot)
				if err != nil {
					res.Failed++
					_, _ = fmt.Fprintf(out, "ล้มเหลว  %s — %s\n", label, seedDescribe(err))
					continue
				}
				if has {
					res.Skipped++
					_, _ = fmt.Fprintf(out, "ข้าม     %s — GLOBAL มีรูปแล้ว (ใช้ -force เพื่อเปลี่ยน)\n", label)
					continue
				}
			}
			data, err := seedReadLimited(path, assetCore.MaxFileSize+1)
			if err != nil {
				res.Failed++
				_, _ = fmt.Fprintf(out, "ล้มเหลว  %s — %v\n", label, err)
				continue
			}
			req := assetDto.UploadRequest{GameCode: g.Name(), AssetType: slot, UseNowValue: true}
			up, err := UploadFileService(ctx, actor, target, req, data)
			if err != nil {
				res.Failed++
				_, _ = fmt.Fprintf(out, "ล้มเหลว  %s — %s\n", label, seedDescribe(err))
				continue
			}
			res.Uploaded++
			_, _ = fmt.Fprintf(out, "อัปแล้ว  %s → %s\n", label, up.URL)
		}
	}

	res.Gaps, err = ListGlobalGapsService(ctx)
	return res, err
}

// seedFilesPerSlot — ชื่อไฟล์รูปของแต่ละช่องในโฟลเดอร์เกม (ชื่อไฟล์ไม่รวมนามสกุล = ชื่อช่อง)
func seedFilesPerSlot(files []os.DirEntry) map[string][]string {
	m := map[string][]string{}
	for _, f := range files {
		if f.IsDir() || !seedImageExts[strings.ToLower(filepath.Ext(f.Name()))] {
			continue
		}
		slot := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
		m[slot] = append(m[slot], f.Name())
	}
	return m
}

func seedReadLimited(path string, limit int64) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // path มาจาก -dir ที่ผู้รันสคริปต์ระบุเอง (เครื่องมือของทีม ไม่ใช่ input จากผู้ใช้)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck
	return io.ReadAll(io.LimitReader(f, limit))
}

// describe — ข้อความของ error ที่ประกาศไว้ (apperr) · อย่างอื่นแสดงตามจริง (สคริปต์รันโดยทีมเอง)
func seedDescribe(err error) string {
	var e *apperr.Error
	if errors.As(err, &e) && e.Code != apperr.ErrInternal.Code {
		return fmt.Sprintf("%d %s", e.Code, e.MsgTH)
	}
	return err.Error()
}
