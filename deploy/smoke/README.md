# token-router process smoke

`token-router-gb10-process-smoke.sh` runs the current gb10-4t real-process proof
from a source checkout:

```bash
deploy/smoke/token-router-gb10-process-smoke.sh
```

It builds fresh binaries into a timestamped directory under `/tmp`, starts a
strict mock supply, seeds a SQLite database, starts the API-only server, runs the
demand simulator, then runs the operations review agent once.

The script preserves its run directory by default. The directory contains:

- `bin/`: built `token-router-api`, `token-router-sim`, `token-router-supply`
- `token-router.db`: smoke SQLite database
- `logs/build.log`: build output
- `logs/mock-supply.log`: mock supply output
- `logs/api.log`: API-only server output
- `logs/demand-sim.log`: demand simulator proof output
- `logs/review-agent.json`: review agent cycle summary
- `summary.txt`: ports and evidence paths

Useful options:

```bash
deploy/smoke/token-router-gb10-process-smoke.sh --run-dir /tmp/tr-smoke
deploy/smoke/token-router-gb10-process-smoke.sh --api-port 19090 --supply-port 19091
deploy/smoke/token-router-gb10-process-smoke.sh --model gpt-test --node-name aima2
```

The smoke runner only touches its temporary run directory and localhost ports.
It does not install systemd units, use production secrets, or perform automatic
approve/apply/activate/disable operations outside the simulator's closed proof
path.

## Enterprise downstream smoke

`enterprise-downstream-customer-smoke.sh` is a quick downstream API-key smoke
against a live service. It covers balance, usage, chat completions, concurrent
requests, invalid-model rejection, and token logs using one supplied downstream
API key.

`enterprise-downstream-live-matrix.py` is the broader live customer matrix. It
logs in with a test user, creates temporary API keys, exercises C-side direct
usage and B-side relay-station usage, checks negative keys, verifies usage/log
accounting, and deletes the temporary keys. It stores only masked keys in the
summary.

```bash
TOKEN_ROUTER_BASE_URL='https://example.com' \
TOKEN_ROUTER_USERNAME='test-user' \
TOKEN_ROUTER_PASSWORD='test-password' \
TOKEN_ROUTER_PRIMARY_MODEL='kimi-test' \
TOKEN_ROUTER_SECONDARY_MODEL='moonlight-16b' \
TOKEN_ROUTER_CONCURRENCY=30 \
TOKEN_ROUTER_C_CONCURRENCY=30 \
deploy/smoke/enterprise-downstream-live-matrix.py
```

`enterprise-b-reseller-c-chain.py` starts temporary local B routers backed by
throwaway SQLite databases, creates A-side upstream keys for each B, creates C
users and multiple C API keys inside every B, then drives concurrent C->B->A
requests and reconciles both ledgers.

```bash
TOKEN_ROUTER_A_BASE_URL='https://example.com' \
TOKEN_ROUTER_A_USERNAME='test-user' \
TOKEN_ROUTER_A_PASSWORD='test-password' \
TOKEN_ROUTER_PRIMARY_MODEL='kimi-test' \
TOKEN_ROUTER_SECONDARY_MODEL='moonlight-16b' \
deploy/smoke/enterprise-b-reseller-c-chain.py \
  --b-count 3 \
  --c-per-b 3 \
  --keys-per-c 3 \
  --requests-per-key 2 \
  --max-workers 54
```
