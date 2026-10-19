# Retired staging deployment

Staging is retired: preserve its data and PostgreSQL database **`centrachannel_staging`**, but keep **no active staging application runtime** on the current server. Production uses `centrachannel` and localhost port 4000; former staging used port 4001.

## Preserved implementation

- `.github/workflows/backend.yml` has been renamed to `.github/workflows/staging.yml` and named **Backend staging CI/CD**. Its existing PR / main / master trigger and staging deployment behavior are retained for reference/future use.
- **Disable `staging.yml` in GitHub Actions**, independently of this rename. Repository variable `STAGING_ENABLED=false` additionally gates the checks job (and its dependent deploy jobs), preventing a first push after the rename from building or deploying. A workflow rename does not carry forward a disabled workflow's identity reliably. Reactivation on the new VM requires both setting `STAGING_ENABLED=true` and explicitly enabling the workflow.
- Keep the old staging Compose file `docker-compose.yml`, `.env.example` staging defaults, existing runtime environment/data, and staging DB. Do not wipe data or remove shared services while retiring the app.
- The production workflow `.github/workflows/production.yml` deploys all backend tags independently. FE tags release only FE in its own repository.

Stopping runtime and disabling Actions are separate operator actions, not side effects of this documentation/code edit. Verify no `centrachannel-staging` runtime remains when retirement is performed. Do not reactivate staging on this VPS: Redis is shared with production without a new prefix or DB index, so simultaneous staging activity is unsafe.

## Future reactivation on a separate VM

1. Provision a separate VM and deliberately restore/reconfigure staging DB, Redis, secrets, network and integration settings. Preserve the old data until a reviewed migration/restore plan exists.
2. Review `staging.yml` before enabling it: it still points to the original checkout path, port 4001, legacy rollback behavior and `STAGING_SSH_*` secrets. Its legacy remote-login command also embeds `GHCR_READ_TOKEN` in SSH command text; replace that approach with a server-side registry login before reuse.
3. Configure the new VM's staging paths, Compose project, domains, proxy and loopback listener. Keep its resources separate from production. Do not copy production secrets.
4. Configure the new staging SSH credentials/known-host secrets, then explicitly enable **Backend staging CI/CD** in GitHub Actions. Confirm the workflow really targets only the new VM before pushing a staging branch.
5. Apply schema changes manually and verify runtime, health, tenant routing and integrations. Neither `entrypoint.sh` nor the production workflow runs migrations/seeds.

There is no active staging-runtime guarantee from YAML alone: operational disablement and runtime retirement must be read back separately by the operator. See [PRODUCTION_DEPLOYMENT.md](PRODUCTION_DEPLOYMENT.md) for current production deployment and rollback.
