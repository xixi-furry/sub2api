import contextlib
import importlib
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

sync = importlib.import_module("sync")

class UpstreamSyncTests(unittest.TestCase):
    def test_only_new_stable_versions_trigger_sync(self):
        self.assertTrue(sync.newer_version("0.2.10", "0.2.9"))
        self.assertFalse(sync.newer_version("0.2.7", "0.2.7"))
        self.assertFalse(sync.newer_version("0.2.6", "0.2.7"))
        with self.assertRaises(ValueError):
            sync.newer_version("0.2.8-rc1", "0.2.7")

    def fixture(self, directory, conflict):
        def git_at(path, *args):
            return subprocess.check_output(["git", "-C", str(path), *args], text=True, stderr=subprocess.STDOUT).strip()
        upstream = directory / "upstream"
        upstream.mkdir()
        git_at(upstream, "init", "-b", "main")
        git_at(upstream, "config", "user.name", "Test")
        git_at(upstream, "config", "user.email", "test@example.invalid")
        (upstream / "app.txt").write_text("original\n")
        git_at(upstream, "add", ".")
        git_at(upstream, "commit", "-m", "upstream base")
        bare = directory / "origin.git"
        subprocess.run(["git", "clone", "--bare", str(upstream), str(bare)], check=True, capture_output=True)
        fork = directory / "fork"
        subprocess.run(["git", "clone", str(bare), str(fork)], check=True, capture_output=True)
        git_at(fork, "config", "user.name", "Test")
        git_at(fork, "config", "user.email", "test@example.invalid")
        (fork / ".github").mkdir()
        (fork / ".github/fork.json").write_text(json.dumps({"upstream_version": "0.2.7"}))
        (fork / "custom.txt").write_text("moderation editors and purple theme\n")
        if conflict:
            (fork / "app.txt").write_text("fork changed this line\n")
        git_at(fork, "add", ".")
        git_at(fork, "commit", "-m", "fork customization")
        git_at(fork, "push", "origin", "main")
        old_main = git_at(bare, "rev-parse", "main")
        (upstream / "app.txt").write_text("official new version\n")
        git_at(upstream, "add", ".")
        git_at(upstream, "commit", "-m", "upstream release")
        git_at(upstream, "tag", "v0.2.8")
        return git_at, upstream, bare, fork, old_main

    def run_merge(self, directory, conflict):
        git_at, upstream, bare, fork, old_main = self.fixture(directory, conflict)
        real_git = sync.git
        def local_fetch(*args, **kwargs):
            args = tuple(str(upstream) if arg == "https://github.com/Wei-Shaw/sub2api.git" else arg for arg in args)
            return real_git(*args, **kwargs)
        release = io.BytesIO(json.dumps({"tag_name": "v0.2.8", "draft": False, "prerelease": False}).encode())
        old_cwd = Path.cwd()
        try:
            os.chdir(fork)
            with patch.dict(os.environ, {
                "GH_TOKEN": "test-only", "GITHUB_RUN_ID": "123", "GITHUB_RUN_ATTEMPT": "1",
                "GITHUB_REPOSITORY": "test/fork", "GITHUB_OUTPUT": str(directory / "output"),
                "GITHUB_STEP_SUMMARY": str(directory / "summary"),
            }), patch.object(sync, "git", local_fetch), patch.object(sync.urllib.request, "urlopen", return_value=release):
                if conflict:
                    with self.assertRaises(subprocess.CalledProcessError), contextlib.redirect_stdout(io.StringIO()):
                        sync.main()
                    self.assertEqual(git_at(fork, "rev-parse", "HEAD"), old_main)
                    self.assertEqual(git_at(fork, "status", "--porcelain"), "")
                    self.assertEqual(git_at(bare, "for-each-ref", "--format=%(refname)", "refs/heads/automation/"), "")
                else:
                    sync.main()
                    branch = "automation/upstream-v0.2.8-123-1"
                    self.assertEqual(git_at(bare, "show", branch + ":custom.txt"), "moderation editors and purple theme")
                    self.assertEqual(git_at(bare, "show", branch + ":app.txt"), "official new version")
                    self.assertIn('"upstream_version": "0.2.8"', git_at(bare, "show", branch + ":.github/fork.json"))
                self.assertEqual(git_at(bare, "rev-parse", "main"), old_main)
        finally:
            os.chdir(old_cwd)

    def test_merge_preserves_customization_and_does_not_touch_main_before_checks(self):
        with tempfile.TemporaryDirectory() as temp:
            self.run_merge(Path(temp), False)

    def test_conflict_aborts_without_changing_main_or_publishing_branch(self):
        with tempfile.TemporaryDirectory() as temp:
            self.run_merge(Path(temp), True)

if __name__ == "__main__":
    unittest.main()
