from pathlib import Path
import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("build_runtime_acceptance.sh")


class RuntimeCompileCommandTests(unittest.TestCase):
    def test_builds_same_commit_two_versions_runner_and_standard_helper_without_tests(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            (root / "scripts").mkdir()
            shutil.copyfile(SCRIPT, root / "scripts" / SCRIPT.name)
            subprocess.run(["git", "init", "-q"], cwd=root, check=True)
            subprocess.run(["git", "add", "scripts"], cwd=root, check=True)
            subprocess.run(["git", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid",
                            "commit", "-qm", "fixture source"], cwd=root, check=True)
            revision = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=root, text=True).strip()
            binary = root / "bin/go"
            binary.parent.mkdir()
            binary.write_text("#!" + sys.executable + "\n"
                              "import json, os, sys\n"
                              "with open(os.environ['GRAFT_GO_RECORD_FILE'], 'a') as stream:\n"
                              "    stream.write(json.dumps({'args': sys.argv[1:], 'cgo': os.environ['CGO_ENABLED'], 'os': os.environ['GOOS'], 'arch': os.environ['GOARCH']}) + '\\n')\n")
            binary.chmod(0o755)
            record = root / "calls.jsonl"
            subprocess.run(["bash", "scripts/build_runtime_acceptance.sh"], cwd=root, check=True,
                           env={**os.environ, "PATH": str(binary.parent) + os.pathsep + os.environ["PATH"],
                                "GRAFT_GO_RECORD_FILE": str(record)})
            calls = [json.loads(line) for line in record.read_text().splitlines()]
            self.assertEqual(len(calls), 5)
            for call in calls:
                self.assertEqual(call["args"][:3], ["-C", str(root / "server"), "build"])
            self.assertIn("version=0.0.0-beta.1", " ".join(calls[0]["args"]))
            self.assertIn("version=0.0.0-beta.2", " ".join(calls[1]["args"]))
            for call in calls[:2]:
                self.assertIn("gitCommit=" + revision, " ".join(call["args"]))
                self.assertIn("gitTreeState=clean", " ".join(call["args"]))
            self.assertEqual(calls[2]["args"][-1], "./cmd/graft-compose-runner")
            self.assertTrue(calls[3]["args"][-1].endswith("/release-source.go"))
            self.assertTrue(calls[4]["args"][-1].endswith("/registry-auth/main.go"))
            self.assertIn("-mod=readonly", calls[4]["args"])
            self.assertTrue(all(call["cgo"] == "0" and call["os"] == "linux" and call["arch"] == "amd64" for call in calls))
            self.assertFalse(any("test" in call["args"] for call in calls))
            (root / "server").mkdir()
            (root / "server/untracked.go").write_text("package ignored")
            rejected = subprocess.run(["bash", "scripts/build_runtime_acceptance.sh"], cwd=root,
                                      capture_output=True, text=True,
                                      env={**os.environ, "PATH": str(binary.parent) + os.pathsep + os.environ["PATH"],
                                           "GRAFT_GO_RECORD_FILE": str(record)})
            self.assertNotEqual(rejected.returncode, 0)
            self.assertIn("clean tracked source", rejected.stderr)
            self.assertEqual(len(record.read_text().splitlines()), 5)
