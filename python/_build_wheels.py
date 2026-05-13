#!/usr/bin/env python3
"""Build platform-specific wheels for shelfctl from GoReleaser archives.

Usage:
    python _build_wheels.py --version 0.4.11 --archives-dir dist/archives --output-dir dist/wheels
"""

import argparse
import hashlib
import os
import stat
import tarfile
import zipfile
from pathlib import Path

PACKAGE_NAME = "shelfctl"

# Maps GoReleaser archive naming to wheel platform tags.
# GoReleaser produces: shelfctl_VERSION_OS_ARCH.{tar.gz,zip}
PLATFORM_MAP = {
    "Darwin_x86_64": {
        "wheel_tag": "macosx_11_0_x86_64",
        "ext": ".tar.gz",
        "binary": "shelfctl",
    },
    "Darwin_arm64": {
        "wheel_tag": "macosx_11_0_arm64",
        "ext": ".tar.gz",
        "binary": "shelfctl",
    },
    "Linux_x86_64": {
        "wheel_tag": "manylinux_2_17_x86_64.manylinux2014_x86_64",
        "ext": ".tar.gz",
        "binary": "shelfctl",
    },
    "Linux_arm64": {
        "wheel_tag": "manylinux_2_17_aarch64.manylinux2014_aarch64",
        "ext": ".tar.gz",
        "binary": "shelfctl",
    },
    "Windows_x86_64": {
        "wheel_tag": "win_amd64",
        "ext": ".zip",
        "binary": "shelfctl.exe",
    },
    "Windows_arm64": {
        "wheel_tag": "win_arm64",
        "ext": ".zip",
        "binary": "shelfctl.exe",
    },
}

INIT_PY = '''\
"""shelfctl - Personal library manager for PDFs using GitHub Release assets."""

__version__ = "{version}"
'''

MAIN_PY = Path(__file__).parent / "shelfctl" / "__main__.py"

METADATA_TEMPLATE = """\
Metadata-Version: 2.1
Name: {name}
Version: {version}
Summary: Personal library manager for PDFs using GitHub Release assets
Home-page: https://github.com/blackwell-systems/shelfctl
Author: Dayna Blackwell
License: MIT
Classifier: License :: OSI Approved :: MIT License
Classifier: Operating System :: MacOS
Classifier: Operating System :: POSIX :: Linux
Classifier: Operating System :: Microsoft :: Windows
Classifier: Environment :: Console
Classifier: Topic :: Utilities
Requires-Python: >=3.8
Description-Content-Type: text/markdown

# shelfctl

CLI tool for organizing PDF and book libraries using GitHub Release assets.

Install: `pip install shelfctl`

Usage: `shelfctl --help`

Full documentation: https://github.com/blackwell-systems/shelfctl
"""

ENTRY_POINTS = """\
[console_scripts]
shelfctl = shelfctl.__main__:main
"""


def sha256_digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def record_hash(data: bytes) -> str:
    """Return the hash string for RECORD: sha256=<urlsafe-base64-no-padding>."""
    import base64

    digest = hashlib.sha256(data).digest()
    return "sha256=" + base64.urlsafe_b64encode(digest).rstrip(b"=").decode()


def extract_binary(archive_path: str, binary_name: str) -> bytes:
    """Extract the binary from a GoReleaser archive."""
    if archive_path.endswith(".tar.gz"):
        with tarfile.open(archive_path, "r:gz") as tf:
            for member in tf.getmembers():
                if os.path.basename(member.name) == binary_name:
                    f = tf.extractfile(member)
                    if f is None:
                        raise ValueError(f"Cannot read {member.name} from {archive_path}")
                    return f.read()
    elif archive_path.endswith(".zip"):
        with zipfile.ZipFile(archive_path, "r") as zf:
            for name in zf.namelist():
                if os.path.basename(name) == binary_name:
                    return zf.read(name)

    raise FileNotFoundError(f"Binary {binary_name} not found in {archive_path}")


def build_wheel(
    version: str,
    platform_key: str,
    platform_info: dict,
    archives_dir: str,
    output_dir: str,
) -> str:
    """Build a single platform wheel. Returns the output path."""
    wheel_tag = platform_info["wheel_tag"]
    binary_name = platform_info["binary"]
    ext = platform_info["ext"]

    # Find the archive
    archive_name = f"{PACKAGE_NAME}_{version}_{platform_key}{ext}"
    archive_path = os.path.join(archives_dir, archive_name)
    if not os.path.exists(archive_path):
        raise FileNotFoundError(f"Archive not found: {archive_path}")

    # Extract binary
    binary_data = extract_binary(archive_path, binary_name)

    # Prepare wheel contents
    dist_info_dir = f"{PACKAGE_NAME}-{version}.dist-info"
    pkg_dir = PACKAGE_NAME

    # File contents
    init_content = INIT_PY.format(version=version).encode()
    main_content = MAIN_PY.read_bytes()
    metadata_content = METADATA_TEMPLATE.format(name=PACKAGE_NAME, version=version).encode()
    wheel_content = (
        f"Wheel-Version: 1.0\n"
        f"Generator: shelfctl-build\n"
        f"Root-Is-Purelib: false\n"
        f"Tag: py3-none-{wheel_tag}\n"
    ).encode()
    entry_points_content = ENTRY_POINTS.encode()

    # Build RECORD
    files = [
        (f"{pkg_dir}/__init__.py", init_content),
        (f"{pkg_dir}/__main__.py", main_content),
        (f"{pkg_dir}/{binary_name}", binary_data),
        (f"{dist_info_dir}/METADATA", metadata_content),
        (f"{dist_info_dir}/WHEEL", wheel_content),
        (f"{dist_info_dir}/entry_points.txt", entry_points_content),
    ]

    record_lines = []
    for path, data in files:
        record_lines.append(f"{path},{record_hash(data)},{len(data)}")
    record_lines.append(f"{dist_info_dir}/RECORD,,")
    record_content = "\n".join(record_lines).encode()

    # Build the wheel zip
    wheel_filename = f"{PACKAGE_NAME}-{version}-py3-none-{wheel_tag}.whl"
    wheel_path = os.path.join(output_dir, wheel_filename)

    with zipfile.ZipFile(wheel_path, "w", compression=zipfile.ZIP_DEFLATED) as whl:
        for path, data in files:
            info = zipfile.ZipInfo(path)
            info.compress_type = zipfile.ZIP_DEFLATED

            # Set executable permission for the binary
            if path == f"{pkg_dir}/{binary_name}":
                info.external_attr = (stat.S_IRWXU | stat.S_IRGRP | stat.S_IXGRP | stat.S_IROTH | stat.S_IXOTH) << 16
            else:
                info.external_attr = (stat.S_IRUSR | stat.S_IWUSR | stat.S_IRGRP | stat.S_IROTH) << 16

            whl.writestr(info, data)

        # Write RECORD last
        record_info = zipfile.ZipInfo(f"{dist_info_dir}/RECORD")
        record_info.compress_type = zipfile.ZIP_DEFLATED
        record_info.external_attr = (stat.S_IRUSR | stat.S_IWUSR | stat.S_IRGRP | stat.S_IROTH) << 16
        whl.writestr(record_info, record_content)

    return wheel_path


def main():
    parser = argparse.ArgumentParser(description="Build platform wheels for shelfctl")
    parser.add_argument("--version", required=True, help="Version string (e.g. 0.4.11)")
    parser.add_argument("--archives-dir", required=True, help="Directory containing GoReleaser archives")
    parser.add_argument("--output-dir", required=True, help="Output directory for .whl files")
    args = parser.parse_args()

    os.makedirs(args.output_dir, exist_ok=True)

    built = []
    for platform_key, platform_info in PLATFORM_MAP.items():
        try:
            path = build_wheel(
                version=args.version,
                platform_key=platform_key,
                platform_info=platform_info,
                archives_dir=args.archives_dir,
                output_dir=args.output_dir,
            )
            built.append(path)
            print(f"  Built: {os.path.basename(path)}")
        except FileNotFoundError as e:
            print(f"  Skip: {platform_key} ({e})")

    print(f"\n{len(built)} wheel(s) built in {args.output_dir}/")


if __name__ == "__main__":
    main()
