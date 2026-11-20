"""Append the game assets as a ZIP readable from the executable itself."""

import pathlib
import sys
import zipfile

exe = pathlib.Path(sys.argv[1])
with zipfile.ZipFile(exe, "a", compression=zipfile.ZIP_DEFLATED, compresslevel=6) as archive:
    for path in sorted(pathlib.Path("assets").rglob("*")):
        if path.is_file() and not any(part.startswith(".") for part in path.parts):
            archive.write(path, path.as_posix())
