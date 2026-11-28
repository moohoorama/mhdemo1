#!/bin/zsh
# 성·마을 장면 맵 생성 전체: 그림 굽기(선택) → 미리보기 루트 → 맵마다 생성 → 렌더 → 걷기 점검
#   tools/towngen/batch.sh            # 그림은 work/reg, work/lines 그대로 쓰고 맵만
#   tools/towngen/batch.sh --art      # 지역 건물·줄 타일 그림부터 다시 굽기 (수 분)
#   tools/towngen/batch.sh S_zhuo:huabei S_ye:huabei   # 일부 맵만
TOOLS=${0:A:h}
VER5=${TOOLS:h:h}
WORK=${TOWNGEN_WORK:-$TOOLS/work}
PY=${PYTHON:-python3}
mkdir -p $WORK/bin $WORK/render/gen $WORK/render/gencur
cd $WORK
ART=0; MAPS=()
for a in "$@"; do [[ $a == --art ]] && ART=1 || MAPS+=$a; done
(( ${#MAPS} )) || MAPS=(S_zhuo:huabei S_pingyuan:huabei S_xindu:huabei S_ye:huabei S_beihai:huabei S_luoyang:zhongyuan S_xudu:zhongyuan S_runan:zhongyuan S_road_town:zhongyuan S_xiaopei:huainan S_xuzhou_town:huainan S_xuzhou_gate:huainan S_xiapi:huainan S_xiapi_inner:huainan S_xiangyang:huainan S_xinye:huainan S_xinye_town:huainan S_mangdang:huainan S_vil_peace:huainan S_jiangling:jiangnan S_jiangxia_harbor:jiangnan S_danyang_village:jiangnan)
(cd $VER5 && go build -o $WORK/bin/maplight ./cmd/_maplight && go build -o $WORK/bin/walkcheck ./cmd/_walkcheck && go build -o $WORK/bin/scenario-admin ./cmd/scenario-admin) || { echo BUILDFAIL; exit 1; }
if (( ART )) || [[ ! -d $WORK/reg ]]; then
  $PY $TOOLS/town_dot2.py $WORK/reg > /dev/null && $PY $TOOLS/town_lines.py $WORK/lines > /dev/null || { echo ARTFAIL; exit 1; }
fi
$PY $TOOLS/build_root.py $WORK/reg $WORK/root $WORK/lines > /dev/null || { echo ROOTFAIL; exit 1; }
for mr in $MAPS; do
  m=${mr%%:*}; r=${mr##*:}
  $PY $TOOLS/towngen.py $WORK/root $m $r > $WORK/gen_$m.log 2>&1 || echo "GENFAIL $m"
  head -1 $WORK/gen_$m.log | $PY -c "import json,sys;d=json.loads(sys.stdin.read());print('$m',d['ok'],'missing',d['missing'],'covered',len(d['covered']),'grow',d['grow'],'fence',d['fenced_cottages'],'placed',{k:v['name'] for k,v in d['placed'].items()})" 2>/dev/null || (echo "PARSEFAIL $m"; tail -3 $WORK/gen_$m.log)
  (cd $WORK/root && $WORK/bin/maplight $m $WORK/render/gen/$m.png) || echo "RENDERFAIL $m"
  [ -f $WORK/render/gencur/$m.png ] || (cd $VER5 && $WORK/bin/maplight $m $WORK/render/gencur/$m.png)
done
$PY $TOOLS/smithy_yard.py $WORK/root huabei zhongyuan huainan > $WORK/smithy_yard.log || echo YARDFAIL
for r in huabei zhongyuan huainan; do (cd $WORK/root && $WORK/bin/maplight S_smithy_yard_$r $WORK/render/gen/S_smithy_yard_$r.png) || echo "RENDERFAIL yard $r"; done
$WORK/bin/walkcheck $WORK/root ${MAPS[@]%%:*} S_smithy_yard_huabei S_smithy_yard_zhongyuan S_smithy_yard_huainan > $WORK/walk_gen.txt; tail -1 $WORK/walk_gen.txt
echo "미리보기 어드민: (cd $WORK/root && $WORK/bin/scenario-admin -root . -addr 127.0.0.1:18766)"
echo BATCHDONE
