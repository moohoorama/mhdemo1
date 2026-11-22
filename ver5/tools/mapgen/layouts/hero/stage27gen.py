#!/usr/bin/env python3
"""Apply the Huarong layout to the authored stage. Run from ver5."""

import json

from southern_battles import HERO, place_huarong

path = HERO / 'stage27.json'
stage = json.loads(path.read_text(encoding='utf-8'))
path.write_text(json.dumps(place_huarong(stage), ensure_ascii=False, indent=1) + '\n',
                encoding='utf-8')
