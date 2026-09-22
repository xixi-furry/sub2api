"""Version planning for stable upstream releases plus numeric fork revisions."""
import argparse
import json
import os
from pathlib import Path
import re
import subprocess

BASE = re.compile(r"^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$")
TAG = re.compile(r"^v((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))-fix([1-9][0-9]*)$")

def next_tag(base, tags):
    if not BASE.fullmatch(base):
        raise ValueError("Expected a stable upstream version such as 0.2.7")
    revisions = [int(m[2]) for tag in tags if (m := TAG.fullmatch(tag)) and m[1] == base]
    return f"v{base}-fix{max(revisions, default=0) + 1}"

def validate_tag(tag, base):
    match = TAG.fullmatch(tag)
    if not match or match[1] != base:
        raise ValueError("Tag must match the tracked upstream version and use -fixN")
    return tag.removeprefix("v")

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--tag", default="")
    args = parser.parse_args()
    base = json.loads(Path(".github/fork.json").read_text())["upstream_version"]
    tags = subprocess.check_output(["git", "tag", "--list"], text=True).splitlines()
    tag = args.tag or next_tag(base, tags)
    version = validate_tag(tag, base)
    sha = subprocess.check_output(["git", "rev-parse", "HEAD"], text=True).strip()
    if tag in tags:
        tagged = subprocess.check_output(["git", "rev-parse", f"refs/tags/{tag}^{{commit}}"], text=True).strip()
        if tagged != sha:
            raise ValueError("Existing release tag points to another commit; it must never be moved")
    outputs = dict(tag=tag, version=version, sha=sha, upstream_version=base)
    with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as stream:
        for key, value in outputs.items():
            stream.write(f"{key}={value}\n")

if __name__ == "__main__":
    main()
