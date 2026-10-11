# Test scaffolding

- `unit/` – place unit tests for services, utils, etc.
- `integration/` – place integration/E2E tests that spin up the full application (database, HTTP server, etc.).

Both directories are added to the repository so CI can discover them even when they are empty.

Run `make test` for unit tests or `make test-all` for all tests. Both disable
the result cache and generate coverage reports in `coverage/`.

PostgreSQL UUID and tenant isolation regressions live in
`database/uuid_schema_test.go`. Set `TEST_DATABASE_URL` to a disposable PostgreSQL
database to include them in either target. They require `psql` and permission to
create schemas; each test creates and removes its own schema. Without this
variable, these database tests are skipped.

```sh
TEST_DATABASE_URL='postgres://test_user:test_password@localhost/test_db?sslmode=disable' make test-all
```
