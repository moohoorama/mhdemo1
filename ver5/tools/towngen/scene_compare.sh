#!/bin/zsh
# 성·마을 맵을 쓰는 스토리 장면을 지금 맵과 새 맵(미리보기 루트)에 놓고 시작·가운데·끝을 그린다 (scenespots).
#   tools/towngen/scene_compare.sh            → $WORK/cap/{old,new}/s.<장면>.{1,2,3}.png, $WORK/cap/pair/<장면>.png
# 코드 사본 둘($WORK/code_old: 저장소 assets, $WORK/code: 미리보기 루트 assets)을 만들어 돌린다. 사본의 scenespots는
# 게임을 하나 시작해 두고 그린다(조건 걸린 걸음이 엔진을 찾다 멈추는 것 방지). 캡처는 하나씩(lockf).
TOOLS=${0:A:h}
VER5=${TOOLS:h:h}
WORK=${TOWNGEN_WORK:-$TOOLS/work}
PY=${PYTHON:-python3}
for c in code code_old; do
  mkdir -p $WORK/$c && rsync -a --delete $VER5/internal $VER5/cmd $WORK/$c/ && cp $VER5/go.mod $VER5/go.sum $WORK/$c/
  perl -0pi -e 's|"srpg/internal/scenes"\n\)|"srpg/internal/scenes"\n\t"srpg/internal/session"\n)|; s|func demoSceneSpots\(d \*Demo\) error \{\n|func demoSceneSpots(d *Demo) error {\n\td.g.request(session.Request{Op: "jump", Seed: 1, Node: d.g.S.Data.Waypoints()[0].Node})\n|' $WORK/$c/internal/gui/uidemo_scenespots.go
done
ln -sfn $WORK/root/assets $WORK/code/assets
ln -sfn $VER5/assets $WORK/code_old/assets
mkdir -p $WORK/cap/old $WORK/cap/new $WORK/cap/pair
MAPS=$(ls $WORK/roles/*.json | xargs -n1 basename | sed 's/\.json$//' | tr '\n' ' ')
$PY - $VER5 $WORK/cap $MAPS <<'PYEOF'
import json, glob, sys
ver5, cap, maps = sys.argv[1], sys.argv[2], set(sys.argv[3:]) | {'S_weapon_shop'}
R = []
for f in sorted(glob.glob(ver5 + '/assets/scenes/ch*.json')):
    for sid, sc in json.load(open(f))['Scenes'].items():
        if sc['Map'] in maps:
            N = sc['Nodes']
            R.append({'Scene': sid, 'Shots': [{'Name': str(i + 1), 'After': N[k]} for i, k in enumerate((0, len(N) // 2, -1))]})
for v in ('old', 'new'):
    json.dump({'Renders': R}, open(f'{cap}/{v}/s.json', 'w'))
print(len(R), 'scenes')
PYEOF
(cd $WORK/code_old && lockf -k $WORK/capture.lock go run ./cmd/_uidemo-vm -ui scenespots -stage $WORK/cap/old/s.json -frames 400 | tail -1)
(cd $WORK/code && lockf -k $WORK/capture.lock go run ./cmd/_uidemo-vm -ui scenespots -stage $WORK/cap/new/s.json -frames 400 | tail -1)
for f in $WORK/cap/new/s.*.1.png; do
  s=${${f:t}#s.}; s=${s%.1.png}
  magick montage $WORK/cap/old/s.$s.{1,2,3}.png $WORK/cap/new/s.$s.{1,2,3}.png -tile 3x2 -geometry +3+3 -background '#222' $WORK/cap/pair/$s.png
done
echo "장면 동선 테스트: (cd $WORK/code && go test -p 1 -count=1 ./internal/gui/ -run 'TestSceneWalksStayOnOpenGround|TestScenesData|TestEventsOnTheWalkDoNotVanishOrPopUp|TestGearGuide')"
echo COMPAREDONE
