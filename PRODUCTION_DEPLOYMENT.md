# Production deployment

## Release contract

- Backend workflow: `.github/workflows/production.yml`, **every pushed tag** (`**`), including tags containing `/`. There is no version filter and no branch-triggered production release.
- FE and BE are separate repositories/releases: a BE tag deploys only the backend. An FE tag deploys only `fe/dist/`.
- GitHub-hosted runners run Go checks and build/push `ghcr.io/handev-id/centrachannel-backend-code:${{ github.sha }}`. Deployment always selects that full commit SHA, never `latest` or a mutable tag. As in the preserved staging pipeline, known baseline Go test failures are currently allowed; vet and changed-file formatting failures block publishing.
- No migration, seed, Redis flush, database reset, image build, or image pruning runs during a normal deployment or container entrypoint.
- **Do not run Go tests/builds, dependency installs, or Docker builds on this 2 CPU / 2 GB server.** Build and CI execution belong on GitHub Actions.

## Layout and isolation

| Item | Production | Retired staging |
|---|---|---|
| Runtime directory | `/home/solodev/centrachannel/centrachannel-production` | BE checkout `/home/solodev/centrachannel/centrachannel-be` |
| Compose project / container | `centrachannel-production` | `centrachannel-staging` |
| Backend listener | `127.0.0.1:4000` (host network) | former `127.0.0.1:4001` |
| PostgreSQL database | `centrachannel` | `centrachannel_staging` (preserved) |
| Public domain (not activated by deployment) | `centrachannel.com` | former `centrachannel.my.id` |

Production root contains `.env`, `docker-compose.yml`, `deploy-production.sh`, protected `.deployment-backup.*` rollback snapshots, and the separately deployed frontend `fe/dist/`. The repository's `docker-compose.production.yml` is installed as runtime `docker-compose.yml`. PostgreSQL, Redis and other shared services remain outside this Compose project. Never use `--remove-orphans` or prune images here.

Redis stays on the existing shared instance and **existing DB setting**, without introducing a prefix or changing `REDIS_DB`. This is not staging/production Redis isolation; staging must remain stopped. The explicitly approved one-time bootstrap `FLUSHALL` affects **all databases on that Redis instance** and must not be repeated in release automation. Coordinate all other consumers before that setup-only operation.

## One-time preparation (operator only)

1. Retire staging runtime and disable its GitHub workflow as described in [STAGING_DEPLOYMENT.md](STAGING_DEPLOYMENT.md), preserving its database/data.
2. Prepare the production root and `fe/dist/`; create a private `.env` (mode `600`). Do not commit it or print its contents. Use production credentials and review all integrations. Required changes from the staging example:

   ```dotenv
   HOST=127.0.0.1
   PORT=4000
   ENV=production
   APP_BASE_DOMAIN=centrachannel.com
   IMAGE_NAME=ghcr.io/handev-id/centrachannel-backend-code
   IMAGE_TAG=<full-40-character-lowercase-commit-SHA>
   DB_NAME=centrachannel
   DB_URL=<production-URL-targeting-centrachannel>
   WEBHOOK_BASE_URL=https://centrachannel.com
   CORS_ALLOWED_ORIGINS=https://centrachannel.com
   ```

   `IMAGE_NAME` and `IMAGE_TAG` must each appear once, unquoted, on their own `KEY=value` lines. Keep `DB_URL` and all `DB_*` fields consistent: the running Go application uses `DB_*`, whereas migration tooling uses `DB_URL`. Preserve the existing Redis connection/DB setting. Review SMTP, storage, Evolution and Meta credentials and tenant domain records; do not blindly copy secrets or demo tenant data. Compose explicitly enforces loopback port 4000 and `ENV=production`.
3. Confirm the separately approved one-time reset targets **only `centrachannel`**, never `centrachannel_staging`. Back up as required, recreate/reset that database, enable `pgcrypto` as PostgreSQL admin (`CREATE EXTENSION IF NOT EXISTS pgcrypto;` on the production database, required for `gen_random_uuid()` on PostgreSQL 12), apply migrations, then seed only after seed-content approval. This document is not authorization for future database resets.
4. **Approved initial seed:** the owner explicitly approved running all of `database/seed.sql`, including demo tenant/users and sample records. These are not secure production accounts; update/remove demo credentials before public activation. The seeded tenant hostname remains `demo.centrachannel.local` until deliberately updated. Future resets/seeds require fresh operator approval.
5. Ensure Docker Compose v2, Bash, Python 3, curl, flock and standard Linux coreutils exist. The deploy SSH user must already be authorized for Docker and have a working GHCR login for private-image pulls. That server-side login already exists; no registry token is sent through an SSH command. Renew it out of band if needed, using password-stdin.
6. Configure production environment/repository secrets: `PRODUCTION_SSH_HOST`, `PRODUCTION_SSH_USER`, `PRODUCTION_SSH_PRIVATE_KEY`, `PRODUCTION_SSH_KNOWN_HOSTS`. Pin the actual host key; do not disable SSH host verification. No `GHCR_READ_TOKEN` is needed by the production workflow. GitHub's built-in token publishes the image with packages-write permission.

### Manual schema operations

Migrations are operator-managed, never an entrypoint or recurring workflow operation. After selecting/pulling the approved SHA and installing the production Compose file, use the image's migration binary with its container environment (do **not** source dotenv as executable shell):

```bash
cd /home/solodev/centrachannel/centrachannel-production
docker compose --env-file .env -p centrachannel-production -f docker-compose.yml pull app
docker compose --env-file .env -p centrachannel-production -f docker-compose.yml run --rm --no-deps --entrypoint /bin/sh app   -c 'exec migrate -path /app/database/migrations -database "$DB_URL" up'
docker compose --env-file .env -p centrachannel-production -f docker-compose.yml run --rm --no-deps --entrypoint /bin/sh app   -c 'exec migrate -path /app/database/migrations -database "$DB_URL" version'
```

For the approved one-time seed, use existing `psql` from the matching tagged source checkout with `database/seed.sql` **and all `database/seeds/` files**, running from that checkout so its `\i database/seeds/...` paths resolve. Provide production DB credentials securely, require `ON_ERROR_STOP` and a single transaction. The runtime image has no `psql` and contains only the seed runner, not the included seed directory; do not attempt to seed from that image. Do not use `make reset` on the production host: Make's dotenv wrapper invokes Go and can auto-install migration tooling.

Later schema changes remain explicit, reviewed operator operations. Keep changes compatible with the rollback image; automatic application rollback never rolls back the database.

## Automated release and rollback

Push a backend tag only when ready for deployment. Actions checks out that tag, builds on GitHub, publishes its commit-SHA image, and transfers only the tagged Compose file and deploy script in a tar archive. It extracts them into a temporary directory and calls:

```bash
bash <release>/scripts/deploy-production.sh <commit-SHA> <release>/docker-compose.production.yml
```

The script locks `.deploy-production.lock` with `flock` before changing production files. It snapshots `.env`, the previous Compose file and deployed script; installs the tagged configuration and root `deploy-production.sh`; updates only `IMAGE_TAG`; then runs:

```bash
docker compose --env-file .env -p centrachannel-production -f docker-compose.yml pull app
docker compose --env-file .env -p centrachannel-production -f docker-compose.yml up -d --no-build app
```

It waits for the selected Compose container to be running/healthy and for `http://127.0.0.1:4000/health` to return `status: ok`. It does not use the public domain as a readiness check. Concurrent Actions production releases are serialized; the host lock also prevents overlapping manual deploys.

On pull/config/startup/health failure, it restores the previous `.env`, Compose file and deployed script. If replacement was attempted, it recreates the previous app with its retained local immutable image (`--pull never`) and checks health again. A first deployment has no prior image/config: on failure its candidate is stopped and the original `.env` restored. Deployment still exits nonzero after successful rollback; a failed rollback is explicitly reported for intervention. SIGINT/SIGTERM trigger rollback, but SIGKILL/host failure cannot be trapped.

Snapshots stay private in `.deployment-backup.*` for manual recovery and contain secrets; do not publish them. Retain previous images and manage snapshot retention separately. To redeploy a previous known-good SHA using the installed configuration:

```bash
cd /home/solodev/centrachannel/centrachannel-production
bash ./deploy-production.sh <previous-full-commit-SHA>
```

For a configuration regression, restore the matching snapshot's `.env`, `docker-compose.yml`, and `deploy-production.sh` under the same lock before recreating the app; do not restore only the tag. Coordinate manual recovery with Actions. Runtime rollback is not a database rollback.

## Caddy and public activation remain separate

`/home/solodev/centrachannel/centrachannel-production/Caddyfile.production.review` is an **inactive review draft**, copied from the active configuration and preserving unrelated sites. Production routing uses `centrachannel.com`, backend `127.0.0.1:4000`, and frontend root `/home/solodev/centrachannel/centrachannel-production/fe/dist`. The draft preserves the active main-domain redirects, `/api*` and `/ws*` proxy routes, and tenant/custom-host catch-all behavior exactly; it does not add a new public `/health` route. It is kept outside Git because the full configuration includes unrelated authentication configuration.

Do not import/apply the draft, reload Caddy, change DNS, or claim public production availability as part of this code change or tag deployment. At this stage only localhost readiness is expected. Later explicitly approved activation must validate the complete Caddy configuration and verify main/tenant/custom-domain TLS, login, API, WebSockets, webhooks and storage. On-demand TLS allows the base domain and exact active registered tenant hostnames; tenant/domain data must use the intended production hosts.

## Verification boundary

Initial bootstrap retired the staging container, preserved its database/data, recreated only `centrachannel`, enabled PostgreSQL 12's `pgcrypto` prerequisite, applied all migrations and explicitly approved demo seeds, and flushed Redis once. Production initially reuses the existing matching immutable backend image and same-origin prebuilt FE assets, without a local build. Verify actual runtime health and GitHub workflow state separately; a source review is not an Actions tag-release run or rollback drill. No local tests/builds or public Caddy activation are authorized as part of this setup.
