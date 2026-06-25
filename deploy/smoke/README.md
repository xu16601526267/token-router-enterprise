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
