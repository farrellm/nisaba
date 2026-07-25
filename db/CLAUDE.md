# Database

Migrations are managed by `golang-migrate`. Local credentials: `nisaba/nisaba/nisaba` (user/password/db).

If `make migrate` reports "no change" but a column is missing, the dev DB's `schema_migrations` version has drifted ahead of the actual schema — recover with `migrate force <version>` then `make migrate`.

Domain tables use `BIGSERIAL` ids and `ON DELETE CASCADE` FKs. String key/value attributes live in normalized child tables with `PRIMARY KEY (parent_id, key)` for uniqueness; free-form `metadata` is a `JSONB` column.
