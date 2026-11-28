#!/bin/zsh
# 성·마을 장면 맵 생성 전체: 그림 굽기(선택) → 미리보기 루트 → 맵마다 생성 → 렌더 → 걷기 점검
#   tools/towngen/batch.sh            # 그림은 work/reg, work/lines 그대로 쓰고 맵만
#   tools/towngen/batch.sh --art      # 지역 건물·줄 타일 그림부터 다시 굽기 (수 분)
#   tools/towngen/batch.sh S_zhuo:huabei S_ye:huabei   # 일부 맵만
TOOLS=${0:A:h}
VER5=${TOOLS:h:h}
WORK=${TOWNGEN_WORK:-$TOOLS/work}
PY=${PYTHON:-python3}
# 출발점: towngen을 게임에 반영하기 전 커밋(f96ad23)의 맵·탐험·장면
BASE_COMMIT=${TOWNGEN_BASE_COMMIT:-f96ad23}
if [[ -z $TOWNGEN_BASE ]]; then
  export TOWNGEN_BASE=$WORK/base/ver5/assets
  if [[ ! -d $TOWNGEN_BASE ]]; then
    mkdir -p $WORK/base && (cd ${VER5:h} && git archive $BASE_COMMIT ver5/assets/graphics/maps ver5/assets/scenario ver5/assets/scenes | tar -x -C $WORK/base)
  fi
fi
mkdir -p $WORK/bin $WORK/render/gen $WORK/render/gencur
cd $WORK
ART=0; MAPS=()
for a in "$@"; do [[ $a == --art ]] && ART=1 || MAPS+=$a; done
(( ${#MAPS} )) || MAPS=(S_zhuo:huabei S_pingyuan:huabei S_xindu:huabei S_ye:huabei S_beihai:huabei S_luoyang:zhongyuan S_xudu:zhongyuan S_runan:zhongyuan S_road_town:zhongyuan S_xiaopei:huainan S_xuzhou_town:huainan S_xuzhou_gate:huainan S_xiapi:huainan S_xiapi_inner:huainan S_xiangyang:huainan S_xinye:huainan S_xinye_town:huainan S_mangdang:huainan S_vil_peace:huainan S_jiangling:jiangnan S_jiangxia_harbor:jiangnan S_danyang_village:jiangnan)
(cd $VER5 && go build -o $WORK/bin/maplight ./cmd/_maplight && go build -o $WORK/bin/walkcheck ./cmd/_walkcheck && go build -o $WORK/bin/scenario-admin ./cmd/scenario-admin) || { echo BUILDFAIL; exit 1; }
if (( ART )) || [[ ! -d $WORK/reg ]]; then
  $PY $TOOLS/town_dot2.py $WORK/reg > /dev/null && $PY $TOOLS/town_lines.py $WORK/lines > /dev/null || { echo ARTFAIL; exit 1; }
fi
$PY $TOOLS/light_props.py $WORK/props > /dev/null || { echo PROPSFAIL; exit 1; }
$PY $TOOLS/build_root.py $WORK/reg $WORK/root $WORK/lines > /dev/null || { echo ROOTFAIL; exit 1; }
for mr in $MAPS; do
  m=${mr%%:*}; r=${mr##*:}
  $PY $TOOLS/towngen.py $WORK/root $m $r > $WORK/gen_$m.log 2>&1 || echo "GENFAIL $m"
  head -1 $WORK/gen_$m.log | $PY -c "import json,sys;d=json.loads(sys.stdin.read());print('$m',d['ok'],'missing',d['missing'],'covered',len(d['covered']),'grow',d['grow'],'fence',d['fenced_cottages'],'placed',{k:v['name'] for k,v in d['placed'].items()})" 2>/dev/null || (echo "PARSEFAIL $m"; tail -3 $WORK/gen_$m.log)
  (cd $WORK/root && $WORK/bin/maplight $m $WORK/render/gen/$m.png) || echo "RENDERFAIL $m"
  [ -f $WORK/render/gencur/$m.png ] || (cd $VER5 && $WORK/bin/maplight $m $WORK/render/gencur/$m.png)
done
$WORK/bin/walkcheck $WORK/root ${MAPS[@]%%:*} S_tool_shop S_tavern S_zhuo_office S_xiaopei_office S_council S_liu_house S_mansion > $WORK/walk_gen.txt; tail -1 $WORK/walk_gen.txt
echo "미리보기 어드민: (cd $WORK/root && $WORK/bin/scenario-admin -root . -addr 127.0.0.1:18766)"
echo BATCHDONE
