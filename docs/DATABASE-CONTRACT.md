# Database contract

The API assumes Microsoft SQL Server and relies on the following database
behaviour. These are deployment prerequisites, not schema definitions.

- Read models are exposed through the configured views or derived queries and
  omit soft-deleted rows.
- Tables provide internal `autoID` values while public API references use the
  business IDs resolved by `repository.Resolver`.
- Every table carries an `AFTER INSERT` trigger that fills the business ID, so
  an `INSERT` may only `OUTPUT` into a table variable. `OUTPUT` without `INTO`
  is rejected outright (Msg 334) on every table in the schema, and the business
  ID column is empty at `OUTPUT` time regardless. The API therefore returns
  `autoID` through `repository.InsertReturningAuto` and re-reads the row through
  a view in the same transaction. Dropping or disabling those triggers changes
  this contract and breaks business ID generation.
- Stock mutations and document posting are performed through the database
  routines expected by the domain repositories; the API adds application locks
  before these calls.
- Foreign-key, check, unique, and business-rule failures must retain enough SQL
  Server error detail for `platform/httpx` to map them to the public error
  envelope.

## Proposals for database owners

1. Version the production schema with reviewed migrations in `deploy/sql/` and
   record the required schema version at application startup.
2. Add supporting indexes for each view/filter/sort combination used by the
   resource descriptors, validating plans against production-sized data.
3. Keep stock consistency inside the stock procedures as the final authority;
   API application locks cannot protect direct database clients.
4. Publish stable SQL error numbers (or a mapping table) for duplicate,
   reference-in-use, insufficient-stock, and business-rule violations.
