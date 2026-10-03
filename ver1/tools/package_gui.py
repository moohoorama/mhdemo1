"""Package a built macOS executable with the runtime assets; no sprite transforms."""
import pathlib,shutil,plistlib
root=pathlib.Path(__file__).resolve().parents[1]
app=root/'bin/SRPG.app/Contents'
(app/'MacOS').mkdir(parents=True,exist_ok=True)
(app/'Resources').mkdir(exist_ok=True)
shutil.copy2(root/'bin/srpg-gui',app/'MacOS/srpg-gui')
shutil.copytree(root/'assets',app/'Resources/assets',dirs_exist_ok=True)
(app/'Info.plist').write_bytes(plistlib.dumps(dict(CFBundleExecutable='srpg-gui',CFBundleIdentifier='local.srpg.opening',CFBundleName='SRPG',CFBundleDisplayName='도원결의 SRPG',CFBundleVersion='1',CFBundlePackageType='APPL',NSHighResolutionCapable=True)))
