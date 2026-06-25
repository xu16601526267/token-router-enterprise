# Token Router systemd deployment

These templates package the current deployable token-router processes for `aima2`
and the later dedicated server:

- `token-router-api.service`: API-only server.
- `token-router-supply-telemetry-agent.service`: resident telemetry loop.
- `token-router-supply-review-agent.service`: resident operations review loop.
- `token-router-supply-telemetry-sweep.timer`: one-shot telemetry sweep timer.
- `token-router-supply-review-once.timer`: one-shot review refresh timer.

Choose one mode per runner:

- Telemetry: enable either `token-router-supply-telemetry-agent.service` or
  `token-router-supply-telemetry-sweep.timer`.
- Review: enable either `token-router-supply-review-agent.service` or
  `token-router-supply-review-once.timer`.

Do not enable both modes for the same runner unless you intentionally want
duplicate API calls.

## Layout

```text
/opt/token-router/bin/token-router-api
/opt/token-router/bin/token-router-supply
/etc/token-router/token-router.env
/var/lib/token-router/
/var/log/token-router/
```

Create the runtime user once:

```bash
sudo useradd --system --home /var/lib/token-router --shell /usr/sbin/nologin token-router
sudo install -d -o token-router -g token-router /opt/token-router/bin /opt/token-router/deploy/systemd /etc/token-router /var/lib/token-router /var/log/token-router
```

Install binaries and units:

```bash
sudo install -m 0755 bin/token-router-api /opt/token-router/bin/token-router-api
sudo install -m 0755 bin/token-router-supply /opt/token-router/bin/token-router-supply
sudo install -m 0644 deploy/systemd/README.md /opt/token-router/deploy/systemd/README.md
sudo install -m 0644 deploy/systemd/*.service deploy/systemd/*.timer /etc/systemd/system/
sudo install -m 0600 deploy/systemd/token-router.env.example /etc/token-router/token-router.env
sudo systemctl daemon-reload
```

Edit `/etc/token-router/token-router.env` before starting services. The
repository example intentionally contains no real admin token.

## Resident mode

```bash
sudo systemctl enable --now token-router-api.service
sudo systemctl enable --now token-router-supply-telemetry-agent.service
sudo systemctl enable --now token-router-supply-review-agent.service
```

## Timer mode

```bash
sudo systemctl enable --now token-router-api.service
sudo systemctl enable --now token-router-supply-telemetry-sweep.timer
sudo systemctl enable --now token-router-supply-review-once.timer
```

## Smoke checks

```bash
systemctl status token-router-api.service
journalctl -u token-router-supply-telemetry-agent.service -n 50 --no-pager
journalctl -u token-router-supply-review-agent.service -n 50 --no-pager
systemctl list-timers 'token-router-supply-*'
```

The supply services only call existing telemetry or review generation APIs. They
do not approve, apply, activate, disable, change routing weights, change prices,
or touch billing, settlement, wallet, payout, invoice, or funds state.
