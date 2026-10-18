#!/bin/sh
# usage: shot.sh scene state out.png [width height]   (headless Chrome screenshot of the viewer)
D=$(cd "$(dirname "$0")/.." && pwd)
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" --headless=new --disable-gpu --hide-scrollbars --virtual-time-budget=${6:-4000} --window-size=${4:-1100},${5:-760} --screenshot="$3" "file://$D/viewer.html#scene=$1&state=$2" >/dev/null 2>&1
