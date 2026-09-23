# WIP

Future and in-progress ideas only. Once something here is finished, delete its entry or move the relevant content into `README.md`/`docs/development.md`/`CLAUDE.md` — don't leave completed items in this file.

## Known coverage gaps (intentional, for now)

- **`main()` itself (`main.go:35-130`, ~59 statements) is untested** because it calls `os.Exit` and `log.Fatal(router.Run(...))`. Covering it would mean extracting a `run() error`; not done since coverage is above target without it.
- **Error branches after `utilities.ValidateTextCharacters`/`ValidatePasswordFormat` are unreachable** (~25 sites across `controllers/`): both use constant regexes, so `regexp.Match` never errors. Dropping the `error` return from those validators would remove the dead branches.
- **`utilities/migrate.go`'s per-table foreign-key remapping (inside `ReplaceValues`'s `secondRun` switch, e.g. `groups` → `values[7]` is the owner column) still encodes an unverified legacy schema.** `MigrateSQL`/`ReplaceValues`/`MigrateDBToV2`'s generic mechanics (two-pass scan, CREATE/INSERT detection, bigint→varchar(100) rewriting, numeric-ID→UUID substitution, the synthesized trailing `ALTER TABLE`, and the panic-on-missing-file behavior) have real tests, using a table name that isn't one of the hardcoded special cases — that sidesteps the one part of this file that depends on knowing the exact old-schema column order for `groups`/`wishes`/`invites`/etc., which isn't documented anywhere and would be a guess. The column indices themselves are still unverified (this was inherited legacy code) and a real legacy-schema dump sample or someone who remembers the old schema would be needed to confirm they're actually right — `TestMigrateSQLColumnMappingOutOfRange` in `migrate_test.go` only proves the *mismatch* is now caught. As a defensive measure, `ReplaceValues` now runs each lookup through `mapColumn`, which bounds-checks the index against the row's actual column count and returns a clear error (surfaced by `MigrateSQL`, and by extension panicking `MigrateDBToV2`, same as its other failure modes) instead of either panicking with a raw out-of-range index or silently substituting the wrong column. Low priority since it's not on any live code path for a current install.

## Planned features

- **Orphaned image cleanup on `/admin`.** Deleting a wish, wishlist or user now removes its image files, but files left behind by deletions made before that change are still on disk. Plan: a section on the admin page (`web/html/admin.html` / `web/js/admin.js`) with two admin-only endpoints under `/api/admin`:
  - A read-only scan listing orphaned files and their total size. That means any `images/wishes/<id>{,_thumbnail}.jpg` whose wish doesn't exist, is disabled, or is on a disabled wishlist, and any `images/profiles/<id>{,_thumbnail}.jpg` whose user doesn't exist or is disabled. Stray `.upload-*.jpg` temp files from interrupted writes count too.
  - A delete action the admin confirms explicitly, which removes exactly what the scan found.

  Keep the scan and the delete as separate steps, since the delete removes user files for good. The filesystem walk belongs in `controllers/image.go`, and the "which of these IDs are still live" lookups in `database/`, which keeps the layering rule.
