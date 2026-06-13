# Test scaffolding

- `unit/` – place unit tests for services, utils, etc.
- `integration/` – place integration/E2E tests that spin up the full application (database, HTTP server, etc.).

Both directories are added to the repository so CI can discover them even when they are empty.
