"""Prepare an upstream release merge without ever resetting the fork's history."""
import json
import os
from pathlib import Path
import re
import subprocess
import urllib.request
from version import BASE

def git(*args, capture=True):
    return subprocess.check_output(["git", *args], text=True).strip() if capture else subprocess.run(["git", *args], check=True)

def newer_version(candidate, current):
    if not BASE.fullmatch(candidate) or not BASE.fullmatch(current):
        raise ValueError("Only stable numeric upstream versions are supported")
    return tuple(map(int, candidate.split("."))) > tuple(map(int, current.split(".")))

def output(**values):
    with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as stream:
        for key, value in values.items():
            stream.write(f"{key}={value}\n")

def main():
    config_path = Path(".github/fork.json")
    config = json.loads(config_path.read_text())
    # Deliberately fixed to the official upstream, never a payload-supplied URL.
    request = urllib.request.Request(
        "https://api.github.com/repos/Wei-Shaw/sub2api/releases/latest",
        headers={"Accept": "application/vnd.github+json", "Authorization": "Bearer " + os.environ["GH_TOKEN"], "User-Agent": "sub2api-fork-sync"},
    )
    with urllib.request.urlopen(request, timeout=30) as response:
        release = json.load(response)
    tag = release["tag_name"]
    candidate = tag.removeprefix("v")
    if release.get("draft") or release.get("prerelease") or not tag.startswith("v") or not BASE.fullmatch(candidate):
        raise ValueError("Upstream latest is not a stable vX.Y.Z release")
    if not newer_version(candidate, config["upstream_version"]):
        output(changed="false")
        print("No newer official release; leaving main and images unchanged.")
        return
    run_id = os.environ["GITHUB_RUN_ID"]
    attempt = os.environ["GITHUB_RUN_ATTEMPT"]
    if not re.fullmatch(r"[0-9]+", run_id) or not re.fullmatch(r"[0-9]+", attempt):
        raise ValueError("Invalid run identifier")
    base = git("rev-parse", "HEAD")
    git("fetch", "--no-tags", "https://github.com/Wei-Shaw/sub2api.git",
        f"refs/tags/{tag}:refs/remotes/upstream/release", capture=False)
    upstream_sha = git("rev-parse", "refs/remotes/upstream/release^{commit}")
    branch = f"automation/upstream-{tag}-{run_id}-{attempt}"
    git("switch", "-c", branch, capture=False)
    git("config", "user.name", "github-actions[bot]", capture=False)
    git("config", "user.email", "41898282+github-actions[bot]@users.noreply.github.com", capture=False)
    try:
        git("merge", "--no-ff", "--no-commit", upstream_sha, capture=False)
    except subprocess.CalledProcessError:
        conflicts = git("diff", "--name-only", "--diff-filter=U")
        print("::error::Upstream merge needs manual resolution. Main was not changed.")
        with open(os.environ["GITHUB_STEP_SUMMARY"], "a", encoding="utf-8") as summary:
            summary.write(f"Merge of {tag} stopped. Main and published images are unchanged.\n\nConflicting files:\n\n\x60\x60\x60\n{conflicts}\n\x60\x60\x60\n")
        git("merge", "--abort", capture=False)
        raise
    config.update(upstream_version=candidate, upstream_commit=upstream_sha)
    config_path.write_text(json.dumps(config, indent=2) + "\n", encoding="utf-8")
    git("add", ".github/fork.json", capture=False)
    git("commit", "-m", f"merge: official {tag}, preserve fork customizations", capture=False)
    sha = git("rev-parse", "HEAD")
    git("push", "origin", f"HEAD:refs/heads/{branch}", capture=False)
    output(changed="true", sha=sha, base=base, branch=branch)
    with open(os.environ["GITHUB_STEP_SUMMARY"], "a", encoding="utf-8") as summary:
        summary.write(f"Candidate branch: [{branch}](https://github.com/{os.environ['GITHUB_REPOSITORY']}/tree/{branch})\n\nMain is only advanced after all required checks succeed.\n")

if __name__ == "__main__":
    main()
