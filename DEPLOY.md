# GoMUD Deployment Runbook

This runbook captures the exact local-to-live deployment steps that worked for the Oracle VPS at `129.80.173.7`.

## Server facts

- SSH user: `ubuntu`
- Host: `129.80.173.7`
- SSH key: `~/Downloads/ssh-key-2026-04-03.key`
- Working tree on VPS: `~/GoMud`
- Live runtime path: `/home/ubuntu/apps/GoMUD`
- Systemd service: `gomud`
- Live binary actually launched by systemd: `/home/ubuntu/apps/GoMUD/gomud`

## Important rules

- Build on the VPS, not on the Mac.
- Do not copy local Mac binaries to the ARM server.
- Keep live users out of sync operations.
- Watch for stale files in the runtime tree. `rsync` can reintroduce renamed/deleted content.
- The web client should use port `8080`, not `80`, unless the service is later changed to run with privilege or behind a reverse proxy.

## Pre-deploy local checks

Run these locally before syncing:

```bash
find ~/projects/GoMud -iname '*base16*' -o -iname '*bace16*'
find ~/projects/GoMud -path '*rooms.instances*'
grep -n 'HttpPort' ~/projects/GoMud/_datafiles/config.yaml
```

Target state:

- Only the correct renamed mob YAML exists.
- No stale room instance overrides are hanging around unintentionally.
- `_datafiles/config.yaml` has `HttpPort: 8080`.

## Sync source to VPS working tree

```bash
rsync -av \
  -e "ssh -i ~/Downloads/ssh-key-2026-04-03.key" \
  --exclude '.git' \
  --exclude 'bin' \
  --exclude '_datafiles/world/default/users' \
  --exclude '_datafiles/world/default/users/**' \
  ~/projects/GoMud/ ubuntu@129.80.173.7:~/GoMud/
```

Notes:

- Sync to `~/GoMud` first, not directly to `/home/ubuntu/apps/GoMUD`.
- Excluding users preserves the live account data on the server.

## SSH to the VPS and build there

```bash
ssh -i ~/Downloads/ssh-key-2026-04-03.key ubuntu@129.80.173.7
cd ~/GoMud
go build -o gomud .
ls -l ~/GoMud/gomud
```

The timestamp on `~/GoMud/gomud` should update.

## Verify critical content/config on VPS before install

```bash
grep -n 'HttpPort' ~/GoMud/_datafiles/config.yaml
find ~/GoMud -iname '*base16*' -o -iname '*bace16*'
```

Target state:

- `HttpPort: 8080`
- No duplicate mob YAMLs like both `100-base16.yaml` and `100-bace16.yaml`

## Optional safety backup

```bash
sudo cp -a /home/ubuntu/apps/GoMUD /home/ubuntu/apps/GoMUD.backup.$(date +%Y%m%d-%H%M%S)
```

## Stop service and install fresh binary

```bash
sudo systemctl stop gomud
sudo install -m 755 ~/GoMud/gomud /home/ubuntu/apps/GoMUD/gomud
```

Important:

- Stop the service first, or you may hit `Text file busy`.
- Use `install`, not a blind overwrite while the process is running.

## Sync content into runtime tree

After the binary is installed, sync the tree into the actual runtime directory:

```bash
sudo rsync -av \
  --exclude '.git' \
  --exclude 'bin' \
  --exclude '_datafiles/world/default/users' \
  --exclude '_datafiles/world/default/users/**' \
  ~/GoMud/ /home/ubuntu/apps/GoMUD/
```

## Re-check runtime tree for known hazards

```bash
find /home/ubuntu/apps/GoMUD -iname '*base16*' -o -iname '*bace16*'
grep -n 'HttpPort' /home/ubuntu/apps/GoMUD/_datafiles/config.yaml
```

Target state:

- Only the correct mob YAML exists in runtime.
- `_datafiles/config.yaml` says `HttpPort: 8080`.

### If duplicate mob files appear

A real failure we hit was:

- `100-base16.yaml`
- `100-bace16.yaml`

Both existed in runtime and both defined `mobid: 100`, which caused startup panic:

- `duplicate id 100 for type *mobs.Mob`

Fix by moving/removing the stale extra YAML before restarting.

## Restart and verify

```bash
sudo systemctl restart gomud
sudo systemctl status gomud --no-pager -l
sudo ss -ltnp | grep -E '(:33333|:44444|:8080|:80)'
```

Expected listeners:

- `33333`
- `44444`
- `8080`

`80` should not be required in the current setup.

## If something fails

Check logs immediately:

```bash
sudo journalctl -u gomud -n 200 --no-pager
```

Known failure signatures:

### Duplicate mob YAML crash

```text
duplicate id 100 for type *mobs.Mob
```

Cause: duplicate mob files in runtime tree.

### Web client not starting

```text
HTTP error=Error starting web server: listen tcp :80: bind: permission denied
```

Cause: config drifted back to `HttpPort: 80`.

Fix:

```bash
sudo sed -i.bak 's/^  HttpPort: 80$/  HttpPort: 8080/' /home/ubuntu/apps/GoMUD/_datafiles/config.yaml
sudo systemctl restart gomud
```

Then verify:

```bash
sudo ss -ltnp | grep -E '(:33333|:44444|:8080)'
```

## Smoke tests

From local machine:

```bash
telnet 129.80.173.7 33333
```

Web client:

- `http://129.80.173.7:8080/`

Recommended in-game checks after deploy:

- login splash looks correct
- hardware puzzle: `connect green white red`
- CTF submission: `submit flag{...}`
- badge recall: `use badge`
- help screen formatting
- Village Hall hints / breadcrumbing

## Short version

```bash
# local
rsync -av -e "ssh -i ~/Downloads/ssh-key-2026-04-03.key" \
  --exclude '.git' \
  --exclude 'bin' \
  --exclude '_datafiles/world/default/users' \
  --exclude '_datafiles/world/default/users/**' \
  ~/projects/GoMud/ ubuntu@129.80.173.7:~/GoMud/

# vps
ssh -i ~/Downloads/ssh-key-2026-04-03.key ubuntu@129.80.173.7
cd ~/GoMud
go build -o gomud .
sudo systemctl stop gomud
sudo install -m 755 ~/GoMud/gomud /home/ubuntu/apps/GoMUD/gomud
sudo rsync -av \
  --exclude '.git' \
  --exclude 'bin' \
  --exclude '_datafiles/world/default/users' \
  --exclude '_datafiles/world/default/users/**' \
  ~/GoMud/ /home/ubuntu/apps/GoMUD/
sudo systemctl restart gomud
sudo systemctl status gomud --no-pager -l
sudo ss -ltnp | grep -E '(:33333|:44444|:8080)'
```
