#!/usr/bin/env bash
# สร้าง tag <env>-<YYYYMMDDHHmm> (เวลาไทย) ที่ปลาย origin/<branch> แล้ว push → .github/workflows/release.yml build image
# ไม่สน branch/commit ในเครื่อง — tag ที่ commit บน origin เสมอ
# รันเอง: make release env=dev | make release env=prod   (ดู docs/DEPLOYMENT.md)
set -euo pipefail
cd "$(dirname "$0")/.."

ENV_NAME=${1:-}
# เพิ่ม env ใหม่: เพิ่ม case ที่นี่ และใน release.yml (on.push.tags + step "tag → branch")
case "$ENV_NAME" in
  dev)  BRANCH=dev ;;
  prod) BRANCH=main ;;
  *) echo "usage: $0 dev|prod"; exit 2 ;;
esac

git fetch --quiet origin "$BRANCH" --tags

# ใช้ ls-tree แทน `git cat-file -e <rev>:<path>` เพราะ Git Bash บน Windows แปลง arg ที่มี ':' + '/' เป็น path Windows
if [ -z "$(git ls-tree --name-only "origin/$BRANCH" .github/workflows/release.yml)" ]; then
  echo "❌ origin/$BRANCH ยังไม่มี .github/workflows/release.yml — tag ไปก็ไม่มี workflow รัน"
  exit 1
fi

TAG="$ENV_NAME-$(TZ=Asia/Bangkok date +%Y%m%d%H%M)"
SHA=$(git rev-parse "origin/$BRANCH")

echo "tag    : $TAG"
echo "commit : $(git log -1 --format='%h %s (%an, %ar)' "$SHA")"

if [ "$ENV_NAME" = prod ]; then
  read -r -p "ยืนยัน tag prod? พิมพ์ yes: " answer
  [ "$answer" = yes ] || { echo "ยกเลิก"; exit 1; }
fi

git tag "$TAG" "$SHA"
if ! git push origin "refs/tags/$TAG"; then
  git tag -d "$TAG" >/dev/null # push ไม่ผ่าน ไม่ให้ tag ค้างในเครื่อง
  exit 1
fi

echo "✅ pushed $TAG → https://github.com/datacenter-projects/mini-game-api/actions/workflows/release.yml"
echo "   image: ghcr.io/datacenter-projects/minigame-api:$TAG"
