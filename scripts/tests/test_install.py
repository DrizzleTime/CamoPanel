"""Exercise the real installer with release fixtures and fake system services.

No Docker daemon, root privileges, network access or host package changes are used.
"""
import hashlib
import io
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]


class InstallerTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / "bin"
        self.bin.mkdir()
        self.systemd = self.root / "systemd"
        self.systemd.mkdir()
        self.target = self.root / "panel"
        self.log = self.root / "commands.log"
        self.release = self.root / "release"
        self.release.mkdir()
        self.make_release()
        self.env = dict(os.environ, PATH=f"{self.bin}:{os.environ['PATH']}",
                        TEST_ROOT=str(self.root), CAMO_ADMIN_PASSWORD="test-password-123")
        self.mock("curl", r'''
url= out=
while (($#)); do
  case "$1" in
    --output) out=$2; shift 2 ;;
    https://*|http://*) url=$1; shift ;;
    *) shift ;;
  esac
done
case "$url" in
  http://127.0.0.1:*/api/health) [[ ${FAIL_HEALTH:-0} != 1 ]]; exit ;;
esac
printf 'download %s\n' "$url" >> "$TEST_ROOT/commands.log"
[[ ${FAIL_DOWNLOAD:-0} != 1 ]] || exit 22
cp "$TEST_ROOT/release/${url##*/}" "$out"
''')
        self.mock("docker", r'''
printf 'docker %s\n' "$*" >> "$TEST_ROOT/commands.log"
if [[ $1 == compose ]]; then
  [[ ${MISSING_COMPOSE:-0} != 1 || -f $TEST_ROOT/compose-installed ]]
elif [[ $1 == info ]]; then
  [[ ${STOPPED_DOCKER:-0} != 1 || -f $TEST_ROOT/docker-started ]]
else
  exit 1
fi
''')
        self.mock("systemctl", r'''
printf 'systemctl %s\n' "$*" >> "$TEST_ROOT/commands.log"
if [[ $* == 'enable --now docker' ]]; then touch "$TEST_ROOT/docker-started"; fi
''')
        self.mock("ss", '[[ ${OCCUPIED_PORT:-0} != 1 ]] || printf "LISTEN 0 128 *:8080 *:*\\n"\n')
        self.mock("sleep", "exit 0\n")
        self.mock("apt-get", r'''
printf 'apt-get %s\n' "$*" >> "$TEST_ROOT/commands.log"
if [[ $* == *install* ]]; then
  touch "$TEST_ROOT/compose-installed" "$TEST_ROOT/docker-installed"
fi
''')
        self.mock("apt-cache", '[[ ${DISTRO_COMPOSE:-0} != 1 ]]\n')
        self.harness = self.root / "run.sh"
        self.harness.write_text(r'''
set -Eeuo pipefail
source "$1"
shift
SYSTEMD_DIR=$TEST_ROOT/systemd
require_root() { :; }
check_system() { ARCH=amd64; }
ensure_tools() { :; }
# Repository configuration needs /etc writes; record that boundary instead.
configure_docker_repository() { printf 'configure-docker-repository\n' >> "$TEST_ROOT/commands.log"; }
command() {
  if [[ ${1:-} == -v && ${2:-} == docker && ${MISSING_DOCKER:-0} == 1 && ! -f $TEST_ROOT/docker-installed ]]; then
    return 1
  fi
  builtin command "$@"
}
main "$@"
''')

    def mock(self, name, body):
        path = self.bin / name
        path.write_text("#!/usr/bin/env bash\nset -euo pipefail\n" + body)
        path.chmod(0o755)

    def make_release(self, version="v0.1.0", corrupt_checksum=False, omit_web=False):
        asset = self.release / "camopanel_linux_amd64.tar.gz"
        files = {"camopanel": (b"#!/bin/sh\nexit 0\n", 0o755),
                 "VERSION": (version.encode() + b"\n", 0o644)}
        if not omit_web:
            files["web/index.html"] = (b"<html>CamoPanel</html>", 0o644)
        with tarfile.open(asset, "w:gz") as archive:
            for name, (content, mode) in files.items():
                member = tarfile.TarInfo(name)
                member.mode = mode
                member.size = len(content)
                archive.addfile(member, io.BytesIO(content))
        checksum = "0" * 64 if corrupt_checksum else hashlib.sha256(asset.read_bytes()).hexdigest()
        (self.release / "checksums.txt").write_text(f"{checksum}  {asset.name}\n")

    def run_installer(self, *args, **env):
        return subprocess.run(
            ["bash", str(self.harness), str(ROOT / "install.sh"), "--yes",
             "--install-dir", str(self.target), *args],
            env={**self.env, **env}, capture_output=True, text=True, timeout=15)

    def commands(self):
        return self.log.read_text() if self.log.exists() else ""

    def assert_no_install(self, result):
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.target.exists())
        self.assertFalse((self.systemd / "camopanel.service").exists())
        self.assertNotIn("apt-get", self.commands())
        self.assertNotIn("docker ", self.commands())

    def test_install_existing_engine_and_keep_password_literal(self):
        password = "  p$word'\"#;$(touch should-not-exist)  "
        result = self.run_installer("--port", "18080", "--username", "owner",
                                    CAMO_ADMIN_PASSWORD=password)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        config = self.target / ".env"
        self.assertEqual(config.stat().st_mode & 0o777, 0o600)
        self.assertIn(f"CAMO_ADMIN_PASSWORD='{password}'", config.read_text())
        self.assertIn("CAMO_ADDR=0.0.0.0:18080", config.read_text())
        self.assertIn("DOCKER_HOST=unix:///var/run/docker.sock", config.read_text())
        self.assertIn("CAMO_ADMIN_USERNAME=owner", config.read_text())
        self.assertEqual((self.target / "VERSION").read_text().strip(), "v0.1.0")
        self.assertTrue((self.target / "web/index.html").exists())
        unit = (self.systemd / "camopanel.service").read_text()
        self.assertIn(f"ExecStart={self.target}/camopanel", unit)
        self.assertIn(f"WorkingDirectory={self.target}", unit)
        self.assertIn("systemctl enable --now camopanel", self.commands())
        self.assertNotIn("apt-get", self.commands())
        self.assertNotIn("systemctl enable --now docker", self.commands())
        self.assertNotIn("configure-docker-repository", self.commands())

    def test_generate_random_password(self):
        result = self.run_installer(CAMO_ADMIN_PASSWORD="")
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertRegex((self.target / ".env").read_text(), r"CAMO_ADMIN_PASSWORD='[a-f0-9]{48}'")

    def test_missing_engine_installs_engine_and_compose(self):
        result = self.run_installer(MISSING_DOCKER="1", MISSING_COMPOSE="1")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("configure-docker-repository", self.commands())
        self.assertIn("docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin", self.commands())

    def test_missing_compose_does_not_reinstall_engine(self):
        result = self.run_installer(MISSING_COMPOSE="1")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("install -y docker-compose-plugin", self.commands())
        self.assertNotIn("configure-docker-repository", self.commands())
        self.assertNotIn("docker-ce-cli", self.commands())

    def test_distro_compose_package(self):
        result = self.run_installer(MISSING_COMPOSE="1", DISTRO_COMPOSE="1")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("install -y docker-compose-v2", self.commands())

    def test_start_stopped_engine(self):
        result = self.run_installer(STOPPED_DOCKER="1")
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn("systemctl enable --now docker", self.commands())

    def test_corrupt_checksum_fails_before_docker_changes(self):
        self.make_release(corrupt_checksum=True)
        result = self.run_installer()
        self.assert_no_install(result)
        self.assertIn("SHA256", result.stderr)

    def test_failed_download_does_not_install(self):
        self.assert_no_install(self.run_installer(FAIL_DOWNLOAD="1"))

    def test_invalid_package_does_not_install(self):
        self.make_release(omit_web=True)
        self.assert_no_install(self.run_installer())

    def test_pinned_version_and_mismatch(self):
        result = self.run_installer("--version", "v0.2.0")
        self.assert_no_install(result)
        self.assertIn("/releases/download/v0.2.0/", self.commands())

    def test_existing_install_is_preserved(self):
        self.target.mkdir()
        config = self.target / ".env"
        config.write_text("original configuration")
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(config.read_text(), "original configuration")
        self.assertEqual(self.commands(), "")

    def test_occupied_port_does_not_install(self):
        self.assert_no_install(self.run_installer(OCCUPIED_PORT="1"))

    def test_invalid_options(self):
        for args in [("--port", "0"), ("--port", "65536"), ("--port", "80"),
                     ("--port", "443"), ("--port", "abc"), ("--version", "../oops"),
                     ("--username", "bad user"), ("--port",), ("--unknown",),
                     ("--install-dir", "/"), ("--install-dir", "/tmp/a%h")]:
            with self.subTest(args=args):
                self.assert_no_install(self.run_installer(*args))

    def test_invalid_password(self):
        for password in ("short", "x" * 73, "password\nline"):
            with self.subTest(password=password):
                self.assert_no_install(self.run_installer(CAMO_ADMIN_PASSWORD=password))

    def test_health_failure_reports_recovery_and_keeps_config(self):
        result = self.run_installer(FAIL_HEALTH="1")
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("journalctl -u camopanel", result.stderr)
        self.assertNotIn("安装完成", result.stdout)
        self.assertTrue((self.target / ".env").exists())

    def test_help_needs_no_privileges(self):
        result = subprocess.run(["bash", str(ROOT / "install.sh"), "--help"],
                                capture_output=True, text=True, timeout=5)
        self.assertEqual(result.returncode, 0)
        self.assertIn("--install-dir", result.stdout)


if __name__ == "__main__":
    unittest.main()
