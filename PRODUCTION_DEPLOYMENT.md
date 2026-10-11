# CentraChannel Production Deployment Guide

This repository currently deploys **staging only**. Production is intentionally not started by the staging workflow.

## Environments

| Environment | Path | Domain | Host/container port | Compose project |
|---|---|---|---:|---|
| Staging | `/home/solodev/centrachannel` | `centrachannel.my.id` | `4001:4001` | `centrachannel-staging` |
| Production | `/home/solodev/centrachannel-prod` | `centrachannel.com` | `4000:4000` | `centrachannel-production` |

Production must have its own `.env`, PostgreSQL database, Redis DB/instance, and tenant data. Do not copy staging secrets or tenant records into production.

## One-time production preparation

1. Create the directory:

   ```bash
   mkdir -p /home/solodev/centrachannel-prod
   ```

2. Copy the Compose file and create a production `.env` from `.env.example`.
3. Set at minimum:

   ```env
   PORT=4000
   HOST=127.0.0.1
   ENV=production
   APP_BASE_DOMAIN=centrachannel.com
   IMAGE_NAME=ghcr.io/handev-id/centrachannel-backend-code
   IMAGE_TAG=<git-sha>
   DB_NAME=centrachannel_production
   REDIS_DB=2
   WEBHOOK_BASE_URL=https://centrachannel.com
   CORS_ALLOWED_ORIGINS=https://centrachannel.com
   ```

4. Add DNS records:

   ```text
   centrachannel.com       A  103.189.235.40
   *.centrachannel.com     A  103.189.235.40
   ```

5. Add a Caddy host matcher for `centrachannel.com` and `*.centrachannel.com` pointing to `127.0.0.1:4000`. Keep the existing staging matcher pointing to `127.0.0.1:4001`.

## Manual migration

Migrations are intentionally not run by container startup or GitHub Actions.

Before deploying a release:

```bash
cd /home/solodev/centrachannel-prod
set -a
. ./.env
set +a
docker compose --project-name centrachannel-production run --rm --entrypoint migrate app \
  -path /app/database/migrations \
  -database "$DB_URL" up
```

Check status with `migrate ... version` before replacing the running container.

## Production deployment

1. Confirm the migration completed successfully.
2. Set `IMAGE_TAG` to the immutable commit SHA in `.env`.
3. Login to GHCR using the dedicated read-only package token.
4. Pull and start:

   ```bash
   docker compose --project-name centrachannel-production pull
   docker compose --project-name centrachannel-production up -d --remove-orphans
   ```

5. Verify:

   ```bash
   curl -fsS https://centrachannel.com/health
   docker compose --project-name centrachannel-production ps
   ```

## Rollback

Keep the previous image SHA. If the new container is unhealthy:

```bash
# Restore the previous IMAGE_TAG in .env, then:
docker compose --project-name centrachannel-production pull

docker compose --project-name centrachannel-production up -d --remove-orphans
curl -fsS https://centrachannel.com/health
```

Do not automatically roll back database migrations. Database migrations must be backward-compatible with the previous application version, or have a separately reviewed down migration.

## Domain cutover

Staging and production use separate databases, so tenant domains do not need an alias migration. For cutover:

1. Prepare production DNS and Caddy first.
2. Prepare the production `.env` and database.
3. Run production migrations manually.
4. Deploy the production image.
5. Verify `/health`, login, tenant subdomain TLS, WebSocket, webhook, and storage.
6. Keep staging online until production verification is complete.

The Caddy On-Demand TLS ask endpoint allows the configured base domain and exact hostnames registered in `domains` where both domain and tenant are active. This includes custom domains outside APP_BASE_DOMAIN. Unknown or inactive names are denied; subdomain suffix alone does not grant permission.

For a custom domain, point DNS at the intended proxy, configure Caddy/proxy routing, then register the full hostname through `/api/domains`. Registration does not create DNS records. Verify TLS, login, API calls, and WebSocket routing through each intended hostname. Every tenant has one active default; other active domains also serve the tenant without redirecting.

JWT tokens identify the tenant UUID and may be supplied on any active hostname for that tenant. Login cookies remain host-specific; signing in on one unrelated custom domain does not automatically sign in on another.

The tenant/domains schema change patches the original development tenant CREATE and adds a separate domains CREATE/DROP migration pair. It is not an upgrade procedure for an existing production database: already-applied versions will not rerun and no data backfill is provided. Prepare a separate reviewed rollout if applying this design to an existing production database; do not reset a production database.
