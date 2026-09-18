#!/usr/bin/env python3
"""Create and verify the portable trusted-backend artifact."""
from __future__ import annotations
import argparse, hashlib, json, pathlib, re, shutil, stat, sys
SHA_RE = re.compile(r"^[0-9a-f]{40}$")
def sha256(path):
    digest=hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024*1024), b""): digest.update(chunk)
    return digest.hexdigest()
def package(binary, output, repository, workflow, run_id, commit):
    commit=commit.lower()
    if not SHA_RE.fullmatch(commit): raise ValueError("commit must be a full 40-character hexadecimal SHA")
    if repository != "gnailuy/sudoku": raise ValueError("repository must be gnailuy/sudoku")
    if not workflow.strip() or run_id <= 0: raise ValueError("workflow and positive run id are required")
    if not binary.is_file() or not binary.stat().st_mode & stat.S_IXUSR: raise ValueError("binary must be an executable regular file")
    if output.exists(): shutil.rmtree(output)
    output.mkdir(parents=True, mode=0o755)
    target=output/"sudoku"; shutil.copyfile(binary,target); target.chmod(0o755)
    manifest={"schema":"sudoku-backend-release/v1","repository":repository,"workflow":workflow,"run_id":run_id,"commit":commit,"executable":"sudoku","sha256":sha256(target)}
    (output/"manifest.json").write_text(json.dumps(manifest,indent=2)+"\n",encoding="utf-8")
def verify(output, expected_commit=None):
    manifest=json.loads((output/"manifest.json").read_text(encoding="utf-8"))
    required={"schema","repository","workflow","run_id","commit","executable","sha256"}
    if set(manifest)!=required or manifest["schema"]!="sudoku-backend-release/v1": raise ValueError("manifest shape or schema is invalid")
    if manifest["repository"]!="gnailuy/sudoku" or not SHA_RE.fullmatch(manifest["commit"]): raise ValueError("manifest identity is invalid")
    if not isinstance(manifest["run_id"],int) or manifest["run_id"]<=0 or not isinstance(manifest["workflow"],str) or not manifest["workflow"].strip(): raise ValueError("manifest workflow identity is invalid")
    if expected_commit and manifest["commit"]!=expected_commit.lower(): raise ValueError("manifest commit does not match expected commit")
    executable=pathlib.PurePosixPath(manifest["executable"])
    if executable.is_absolute() or ".." in executable.parts or executable.parts!=("sudoku",): raise ValueError("manifest executable path is unsafe")
    binary=output/executable
    if not binary.is_file() or not binary.stat().st_mode & stat.S_IXUSR: raise ValueError("artifact binary is missing or not executable")
    if sha256(binary)!=manifest["sha256"]: raise ValueError("artifact checksum mismatch")
def main():
    parser=argparse.ArgumentParser(); subs=parser.add_subparsers(dest="command",required=True)
    create=subs.add_parser("create")
    for name,kwargs in (("--binary",{"type":pathlib.Path,"required":True}),("--output",{"type":pathlib.Path,"required":True}),("--repository",{"required":True}),("--workflow",{"required":True}),("--run-id",{"type":int,"required":True}),("--commit",{"required":True})): create.add_argument(name,**kwargs)
    check=subs.add_parser("verify"); check.add_argument("--output",type=pathlib.Path,required=True); check.add_argument("--commit")
    args=parser.parse_args()
    if args.command=="create": package(args.binary,args.output,args.repository,args.workflow,args.run_id,args.commit); verify(args.output,args.commit)
    else: verify(args.output,args.commit)
    print(json.dumps({"status":"ok","command":args.command}))
if __name__=="__main__":
    try: main()
    except (OSError,ValueError,json.JSONDecodeError) as error:
        print(f"release artifact error: {error}",file=sys.stderr); raise SystemExit(1) from error
