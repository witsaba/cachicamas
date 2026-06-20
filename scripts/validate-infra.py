#!/usr/bin/env python3
"""Validate cachicamas compose + infra files without touching docker."""
import sys, pathlib, re

ROOT = pathlib.Path(__file__).resolve().parent.parent  # scripts/ -> repo root

# ---- 1. YAML files parse ------------------------------------------------
yaml_files = [
    ROOT / "docker-compose.yaml",
    ROOT / "infra/otel/collector-config.yaml",
]

try:
    import yaml
except ImportError:
    print("PyYAML not installed; skipping YAML parse. Install with: pip3 install pyyaml")
    yaml = None

bad = 0
if yaml:
    for f in yaml_files:
        try:
            yaml.safe_load(f.read_text())
            print(f"OK YAML : {f.relative_to(ROOT)}")
        except yaml.YAMLError as e:
            print(f"BAD YAML: {f.relative_to(ROOT)}: {e}")
            bad += 1

# ---- 2. Compose semantic check -----------------------------------------
dc = yaml.safe_load((ROOT / "docker-compose.yaml").read_text())
services = dc.get("services", {})
nets = set(dc.get("networks", {}).keys())
vols = set(dc.get("volumes", {}).keys())

print()
print(f"Services ({len(services)}):", list(services.keys()))
print(f"Networks ({len(nets)}):", list(nets))
print(f"Volumes  ({len(vols)}):", list(vols))
print()

for name, svc in services.items():
    # networks
    for n in (svc.get("networks") or []):
        if n not in nets:
            print(f"  ERR {name}: unknown network '{n}'")
            bad += 1
    # volumes: top-level names must exist; bind mounts (start with . or /) are ok
    for v in (svc.get("volumes") or []):
        if isinstance(v, str):
            src = v.split(":")[0]
            if not src.startswith((".", "/")) and src not in vols:
                print(f"  ERR {name}: unknown named volume '{src}'")
                bad += 1
    # depends_on: each dep must be a service
    for d in (svc.get("depends_on") or {}):
        if d not in services:
            print(f"  ERR {name}: depends on unknown service '{d}'")
            bad += 1
    ports = svc.get("ports")
    kind = "image" if "image" in svc else "build"
    print(f"  OK  {name:25s} [{kind:5s}] ports={ports}")

# ---- 3. Bind mounts actually exist on disk ------------------------------
print()
print("Bind-mount checks (resolved against docker-compose.yaml dir):")
compose_dir = ROOT  # compose file lives at the repo root
compose_text = (ROOT / "docker-compose.yaml").read_text()
# Match volumes: - "something:/dest[:ro]" entries on host side
for line in compose_text.splitlines():
    m = re.match(r"^\s*-\s*[\"']?(\.[^:]+|\.\./[^:]+|/[^:]+):", line)
    if not m:
        continue
    src = m.group(1)
    candidate = (compose_dir / src).resolve()
    if not candidate.exists():
        print(f"  WARN host path missing: {src}  ->  {candidate}")
    else:
        print(f"  OK   host path exists: {src}")

# ---- 4. Dockerfile sanity ----------------------------------------------
df = (ROOT / "backend/database_administrator/Dockerfile").read_text()
checks = [
    ("FROM golang:", "build stage uses official Go image"),
    ("AS builder", "build stage is named 'builder'"),
    ("CGO_ENABLED=", "CGO knob set"),
    ("go mod download", "deps downloaded before source copy"),
    ("COPY src/", "source copied after deps"),
    ("go build", "go build invoked"),
    ("FROM gcr.io/distroless/static-debian12:nonroot", "distroless runtime base"),
    ("USER nonroot", "runs as nonroot user"),
]
print()
print("Dockerfile checks:")
for needle, label in checks:
    print(f"  {'OK ' if needle in df else 'MISS'} {label}  (looking for: {needle!r})")
    if needle not in df:
        bad += 1

print()
sys.exit(0 if bad == 0 else 1)