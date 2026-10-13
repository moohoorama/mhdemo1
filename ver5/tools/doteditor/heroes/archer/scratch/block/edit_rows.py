from pathlib import Path
import sys
p=Path(sys.argv[1]); rows=p.read_text().splitlines(); edits=sys.stdin.read().splitlines()
for row in edits:
    if not row: continue
    number, pixels=row.split('|',1)
    assert len(pixels)==56,(number,len(pixels))
    index=next(i for i,line in enumerate(rows) if line.startswith(number+'|'))
    rows[index]=row
p.write_text('\n'.join(rows)+'\n')
