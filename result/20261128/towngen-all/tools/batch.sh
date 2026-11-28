#!/bin/zsh
# 성·마을 맵 전부: 루트 새로 → 맵마다 생성 → 렌더 → 걷기 점검
SP=/private/tmp/claude-501/-Users-younwookim-orca-mhdemo1-ver5/2c113d85-85bd-4d6c-8a74-3c6034a9faac/scratchpad
cd $SP
/Users/younwookim/.pyenv/versions/3.11.2/bin/python3 build_root.py reg root lines > /dev/null || { echo ROOTFAIL; exit 1; }
mkdir -p render/gen render/gencur
MAPS=(S_zhuo:huabei S_pingyuan:huabei S_xindu:huabei S_ye:huabei S_beihai:huabei S_luoyang:zhongyuan S_xudu:zhongyuan S_runan:zhongyuan S_road_town:zhongyuan S_xiaopei:huainan S_xuzhou_town:huainan S_xuzhou_gate:huainan S_xiapi:huainan S_xiapi_inner:huainan S_xiangyang:huainan S_xinye:huainan S_xinye_town:huainan S_mangdang:huainan S_vil_peace:huainan S_jiangling:jiangnan S_jiangxia_harbor:jiangnan S_danyang_village:jiangnan)
for mr in $MAPS; do
  m=${mr%%:*}; r=${mr##*:}
  /Users/younwookim/.pyenv/versions/3.11.2/bin/python3 towngen.py root $m $r > gen_$m.log 2>&1 || echo "GENFAIL $m"
  head -1 gen_$m.log | /Users/younwookim/.pyenv/versions/3.11.2/bin/python3 -c "import json,sys;d=json.loads(sys.stdin.read());print('$m',d['ok'],'missing',d['missing'],'covered',len(d['covered']),'grow',d['grow'],'fence',d['fenced_cottages'],'placed',{k:v['name'] for k,v in d['placed'].items()})" 2>/dev/null || (echo "PARSEFAIL $m"; tail -3 gen_$m.log)
  (cd root && $SP/bin/maplight $m $SP/render/gen/$m.png) || echo "RENDERFAIL $m"
  [ -f render/gencur/$m.png ] || (cd /Users/younwookim/orca/mhdemo1/ver5 && $SP/bin/maplight_before $m $SP/render/gencur/$m.png)
done
$SP/bin/walkcheck $SP/root ${MAPS[@]%%:*} > walk_gen.txt; tail -1 walk_gen.txt
echo BATCHDONE
