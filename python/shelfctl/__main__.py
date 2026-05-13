"""Thin entry point that exec's the shelfctl binary."""

import os
import subprocess
import sys


def _find_binary() -> str:
    """Locate the shelfctl binary bundled in this package."""
    package_dir = os.path.dirname(os.path.abspath(__file__))

    if sys.platform == "win32":
        binary_name = "shelfctl.exe"
    else:
        binary_name = "shelfctl"

    candidate = os.path.join(package_dir, binary_name)
    if os.path.isfile(candidate):
        # Ensure the binary is executable (pip may not preserve permissions)
        if sys.platform != "win32" and not os.access(candidate, os.X_OK):
            os.chmod(candidate, 0o755)
        return candidate

    raise FileNotFoundError(
        f"Could not find the shelfctl binary. Searched: {candidate}"
    )


def main() -> None:
    binary = _find_binary()

    if sys.platform == "win32":
        completed = subprocess.run([binary] + sys.argv[1:])
        sys.exit(completed.returncode)
    else:
        os.execvp(binary, [binary] + sys.argv[1:])


if __name__ == "__main__":
    main()
