#!/usr/bin/env python3
"""Build an installable debug APK with Go, a JDK, and the Android SDK/NDK."""

from __future__ import annotations

import argparse
import hashlib
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile
import zipfile


ROOT = Path(__file__).resolve().parents[1]


def run(*args: str | Path, env: dict[str, str]) -> None:
    command = [str(arg) for arg in args]
    print("+ " + " ".join(command), flush=True)
    subprocess.run(command, cwd=ROOT, env=env, check=True)


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sdk", type=Path, default=Path(os.environ.get(
        "ANDROID_HOME", os.environ.get("ANDROID_SDK_ROOT", str(Path.home() / "Library/Android/sdk")))))
    parser.add_argument("--api", type=int, default=33, help="installed compile SDK (default: 33)")
    parser.add_argument("--targets", default="android/arm64,android/amd64")
    parser.add_argument("--output", type=Path, default=ROOT / "bin/android/srpg-ver5-debug.apk")
    args = parser.parse_args()
    sdk = args.sdk.expanduser().resolve()
    android_jar = sdk / f"platforms/android-{args.api}/android.jar"
    if not android_jar.is_file():
        parser.error(f"Install Android SDK Platform {args.api}: {android_jar}")
    versions = [p for p in (sdk / "build-tools").iterdir()
                if p.is_dir() and re.fullmatch(r"\d+(\.\d+)+", p.name)]
    if not versions:
        parser.error("Install Android SDK Build-Tools")
    build_tools = max(versions, key=lambda p: tuple(int(n) for n in p.name.split(".")))
    for tool in ("aapt2", "d8", "zipalign", "apksigner"):
        if not (build_tools / tool).is_file():
            parser.error(f"Missing Android build tool: {tool}")
    for tool in ("go", "java", "javac", "keytool"):
        if not shutil.which(tool):
            parser.error(f"Missing executable: {tool}")

    out = ROOT / "bin/android"
    tool_dir = out / "tools"
    tool_dir.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, ANDROID_HOME=str(sdk), GOBIN=str(tool_dir))
    env["PATH"] = str(tool_dir) + os.pathsep + env["PATH"]
    ebiten = re.search(r"github.com/hajimehoshi/ebiten/v2\s+(\S+)", (ROOT / "go.mod").read_text())
    gomobile = re.search(r"github.com/ebitengine/gomobile\s+(\S+)", (ROOT / "go.mod").read_text())
    assert ebiten and gomobile
    # Installing versioned tools avoids changing the game's go.mod/go.sum.
    run("go", "install", f"github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@{ebiten[1]}", env=env)
    run("go", "install", f"github.com/ebitengine/gomobile/cmd/gomobile@{gomobile[1]}",
        f"github.com/ebitengine/gomobile/cmd/gobind@{gomobile[1]}", env=env)
    run(tool_dir / "gomobile", "init", env=env)
    aar = out / "srpg.aar"
    run(tool_dir / "ebitenmobile", "bind", "-target", args.targets, "-androidapi", "23",
        "-javapkg", "com.orca.srpg", "-trimpath",
        "-ldflags=-s -w -extldflags=-Wl,-z,max-page-size=16384", "-o", aar, "./mobile", env=env)

    args.output = args.output.expanduser().resolve()
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(prefix="srpg-apk-") as temporary:
        work = Path(temporary)
        unpacked = work / "aar"
        with zipfile.ZipFile(aar) as archive:
            archive.extractall(unpacked)
        assets = work / "assets"
        assets.mkdir()
        with zipfile.ZipFile(assets / "runtime.zip", "w", zipfile.ZIP_DEFLATED) as archive:
            for path in sorted((ROOT / "assets").rglob("*")):
                if path.is_file() and not any(part.startswith(".") for part in path.relative_to(ROOT).parts):
                    archive.write(path, path.relative_to(ROOT))
        revision = hashlib.sha256((assets / "runtime.zip").read_bytes()).hexdigest()
        (assets / "runtime.sha256").write_text(revision, encoding="ascii")
        unsigned = work / "unsigned.apk"
        run(build_tools / "aapt2", "link", "-I", android_jar, "--manifest",
            ROOT / "android/AndroidManifest.xml", "-A", assets, "-o", unsigned, env=env)
        classes = work / "classes"
        classes.mkdir()
        run("javac", "--release", "8", "-classpath",
            str(android_jar) + os.pathsep + str(unpacked / "classes.jar"), "-d", classes,
            *sorted((ROOT / "android/src").rglob("*.java")), env=env)
        app_jar = work / "app.jar"
        with zipfile.ZipFile(app_jar, "w") as archive:
            for path in classes.rglob("*.class"):
                archive.write(path, path.relative_to(classes))
        dex = work / "dex"
        dex.mkdir()
        run(build_tools / "d8", "--min-api", "23", "--lib", android_jar,
            "--output", dex, app_jar, unpacked / "classes.jar", env=env)
        with zipfile.ZipFile(unsigned, "a", zipfile.ZIP_DEFLATED) as archive:
            for path in sorted(dex.glob("*.dex")):
                archive.write(path, path.name)
            for path in sorted((unpacked / "jni").rglob("*.so")):
                archive.write(path, Path("lib") / path.relative_to(unpacked / "jni"))
        aligned = work / "aligned.apk"
        run(build_tools / "zipalign", "-f", "4", unsigned, aligned, env=env)
        key = out / "debug.keystore"
        if not key.exists():
            run("keytool", "-genkeypair", "-keystore", key, "-storepass", "android",
                "-keypass", "android", "-alias", "androiddebugkey", "-keyalg", "RSA",
                "-keysize", "2048", "-validity", "10000", "-dname",
                "CN=Android Debug,O=Android,C=US", env=env)
        run(build_tools / "apksigner", "sign", "--ks", key, "--ks-pass", "pass:android",
            "--key-pass", "pass:android", "--out", args.output, aligned, env=env)
        run(build_tools / "apksigner", "verify", "--verbose", args.output, env=env)
    print(f"APK: {args.output} ({args.output.stat().st_size / 1024 / 1024:.1f} MiB)")


if __name__ == "__main__":
    main()
