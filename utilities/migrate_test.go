package utilities

import (
	"bufio"
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestChangeColumns(t *testing.T) {
	cases := []struct {
		name    string
		table   string
		line    string
		want    string
		wantOld string // a substring that must be gone from the result
	}{
		{"groups renames owner", "groups", "  `owner` bigint(20) UNSIGNED,", "`owner_id`", "`owner`"},
		{"group_memberships renames member and group", "group_memberships", "`member`,`group`", "`member_id`", "`member`,"},
		{"invites renames invite_* columns", "invites", "`invite_code`,`invite_used`,`invite_recipient`,`invite_enabled`", "`code`", "`invite_code`"},
		{"wishes renames owner and wishlist", "wishes", "`owner`,`wishlist`", "`owner_id`", "`owner`,"},
		{"wishlists renames owner", "wishlists", "`owner`", "`owner_id`", ""},
		{"wishlist_collaborators renames user and wishlist", "wishlist_collaborators", "`user`,`wishlist`", "`user_id`", "`user`,"},
		{"wishlist_memberships renames group and wishlist", "wishlist_memberships", "`group`,`wishlist`", "`group_id`", "`group`,"},
		{"wish_claims renames wish and user", "wish_claims", "`wish`,`user`", "`wish_id`", "`wish`,"},
		{"unknown table leaves column names untouched", "some_other_table", "`owner`", "`owner`", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ChangeColumns(c.line, c.table)
			if want := c.want; want != "" {
				if !strings.Contains(got, want) {
					t.Errorf("ChangeColumns(%q, %q) = %q, want it to contain %q", c.line, c.table, got, want)
				}
			}
		})
	}

	t.Run("bigint columns always become varchar(100)", func(t *testing.T) {
		got := ChangeColumns("`id` bigint(20) UNSIGNED NOT NULL,", "wishes")
		if !strings.Contains(got, "varchar(100)") {
			t.Errorf("ChangeColumns() = %q, want bigint replaced with varchar(100)", got)
		}
		if strings.Contains(got, "bigint") {
			t.Errorf("ChangeColumns() = %q, want no remaining bigint reference", got)
		}
	})

	t.Run("plain bigint without UNSIGNED also becomes varchar(100)", func(t *testing.T) {
		got := ChangeColumns("`ref` bigint(20),", "wishes")
		if !strings.Contains(got, "varchar(100)") {
			t.Errorf("ChangeColumns() = %q, want bigint replaced with varchar(100)", got)
		}
	})
}

func TestMatchIDToUUID(t *testing.T) {
	maps := []IDMap{
		{TableName: "users", ID: "1", UUID: "uuid-for-user-1"},
		{TableName: "wishlists", ID: "1", UUID: "uuid-for-wishlist-1"},
	}

	t.Run("NULL passes through unchanged", func(t *testing.T) {
		if got := MatchIDToUUID(maps, "users", "NULL"); got != "NULL" {
			t.Errorf("MatchIDToUUID(NULL) = %q, want NULL", got)
		}
	})

	t.Run("known ID resolves to its mapped UUID", func(t *testing.T) {
		got := MatchIDToUUID(maps, "users", "1")
		want := "'uuid-for-user-1'"
		if got != want {
			t.Errorf("MatchIDToUUID = %q, want %q", got, want)
		}
	})

	t.Run("same ID in a different table does not match", func(t *testing.T) {
		got := MatchIDToUUID(maps, "groups", "1")
		if got == "'uuid-for-user-1'" || got == "'uuid-for-wishlist-1'" {
			t.Errorf("MatchIDToUUID matched the wrong table's mapping: %q", got)
		}
	})

	t.Run("unknown ID falls back to a freshly generated UUID", func(t *testing.T) {
		got := MatchIDToUUID(maps, "users", "999")
		if len(got) < 2 || got[0] != '\'' || got[len(got)-1] != '\'' {
			t.Errorf("MatchIDToUUID fallback = %q, want a quoted UUID", got)
		}
	})
}

// TestMigrateSQLGenericTable exercises MigrateSQL's real parsing/ID-rewrite
// mechanics end to end using a table name that ISN'T one of the hardcoded
// special cases in ChangeColumns/ReplaceValues (groups, wishes, ...). That
// sidesteps the one part of this legacy parser that depends on knowing the
// exact old-schema column order for those specific tables (undocumented,
// and risky to guess at - see docs/wip.md) while still genuinely covering
// the generic line-by-line scan, the two-pass ID->UUID substitution, and the
// trailing ALTER TABLE synthesis, which is the bulk of what this file does.
func TestMigrateSQLGenericTable(t *testing.T) {
	input := "CREATE TABLE `widgets` (\n" +
		"`id` bigint(20) UNSIGNED NOT NULL,\n" +
		"`name` varchar(255) NOT NULL\n" +
		");\n" +
		"INSERT INTO `widgets` (`id`, `name`) VALUES\n" +
		"(1, 'First'),\n" +
		"(2, 'Second');\n" +
		"\n"

	scanner := bufio.NewScanner(strings.NewReader(input))
	result, err := MigrateSQL(scanner)
	if err != nil {
		t.Fatalf("MigrateSQL returned error: %v", err)
	}

	if !strings.Contains(result, "varchar(100)") || strings.Contains(result, "bigint") {
		t.Errorf("expected the bigint id column to become varchar(100); got:\n%s", result)
	}
	if strings.Contains(result, "(1, 'First')") || strings.Contains(result, "(2, 'Second')") {
		t.Errorf("expected the numeric primary keys to be replaced with UUIDs; got:\n%s", result)
	}

	uuidRe := regexp.MustCompile(`'[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}'`)
	if matches := uuidRe.FindAllString(result, -1); len(matches) != 2 {
		t.Errorf("expected exactly 2 UUID-shaped primary keys (one per row), got %d in:\n%s", len(matches), result)
	}

	if !strings.Contains(result, "ALTER TABLE `widgets`") ||
		!strings.Contains(result, "ADD PRIMARY KEY (`id`)") ||
		!strings.Contains(result, "ADD KEY `idx_widgets_deleted_at` (`deleted_at`)") {
		t.Errorf("expected a synthesized PRIMARY KEY/index ALTER TABLE for widgets; got:\n%s", result)
	}
	if !strings.HasSuffix(strings.TrimSpace(result), "COMMIT;") {
		t.Errorf("expected the output to end with COMMIT;, got:\n%s", result)
	}
}

// TestMigrateSQLStopsAtAlterTable confirms the scan stops as soon as it hits
// a line starting with "ALTER TABLE" in the source dump, rather than trying
// to migrate the original (pre-migration) trailing ALTER TABLE statements a
// real mysqldump file ends with.
func TestMigrateSQLStopsAtAlterTable(t *testing.T) {
	input := "CREATE TABLE `widgets` (\n" +
		"`id` bigint(20) UNSIGNED NOT NULL\n" +
		");\n" +
		"ALTER TABLE `widgets`\n" +
		"\tADD PRIMARY KEY (`id`);\n"

	scanner := bufio.NewScanner(strings.NewReader(input))
	result, err := MigrateSQL(scanner)
	if err != nil {
		t.Fatalf("MigrateSQL returned error: %v", err)
	}
	// The original ALTER TABLE line/index name must not survive verbatim -
	// only the synthesized one (which uses idx_widgets_deleted_at) should.
	if strings.Contains(result, "ADD PRIMARY KEY (`id`);\n\tADD PRIMARY KEY") {
		t.Errorf("expected the original trailing ALTER TABLE block to be dropped, got:\n%s", result)
	}
}

// TestMigrateSQLColumnMappingOutOfRange covers the defensive guard in
// mapColumn: the per-table column-index mapping in ReplaceValues (e.g.
// "groups" expects an owner ID at column index 7) encodes an undocumented,
// unverified legacy schema (see docs/wip.md). If a real dump's row for one of
// those special-cased tables has fewer columns than assumed, MigrateSQL must
// return a clear error identifying the mismatch instead of panicking with an
// unhelpful out-of-range index or silently rewriting the wrong column.
func TestMigrateSQLColumnMappingOutOfRange(t *testing.T) {
	input := "CREATE TABLE `groups` (\n" +
		"`id` bigint(20) UNSIGNED NOT NULL\n" +
		");\n" +
		"INSERT INTO `groups` (`id`, `name`, `owner`) VALUES\n" +
		"(1, 'Family', 5);\n" +
		"\n"

	scanner := bufio.NewScanner(strings.NewReader(input))
	_, err := MigrateSQL(scanner)
	if err == nil {
		t.Fatal("expected MigrateSQL to return an error for a row too short for the groups column mapping")
	}
	if !strings.Contains(err.Error(), "groups") {
		t.Errorf("error = %q, want it to name the table", err.Error())
	}
}

func TestMigrateDBToV2(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := os.MkdirAll("files", 0755); err != nil {
			t.Fatalf("failed to create files dir: %v", err)
		}
		input := "CREATE TABLE `widgets` (\n`id` bigint(20) UNSIGNED NOT NULL\n);\nINSERT INTO `widgets` (`id`) VALUES\n(1);\n\n"
		if err := os.WriteFile("files/db.sql", []byte(input), 0644); err != nil {
			t.Fatalf("failed to write db.sql: %v", err)
		}

		MigrateDBToV2()

		out, err := os.ReadFile("files/db_modified_sql_file.sql")
		if err != nil {
			t.Fatalf("expected db_modified_sql_file.sql to be written: %v", err)
		}
		if !strings.Contains(string(out), "varchar(100)") {
			t.Errorf("expected the migrated output to contain the rewritten column type, got:\n%s", out)
		}
	})

	t.Run("missing source file panics", func(t *testing.T) {
		t.Chdir(t.TempDir())
		// No ./files directory at all, so os.Open fails and MigrateDBToV2
		// panics by design (it's a manual, one-shot ops tool).
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MigrateDBToV2 to panic when ./files/db.sql is missing")
			}
		}()
		MigrateDBToV2()
	})
}

// TestMigrateSQLQuotedValueContainingSeparator covers the re-joining of a
// quoted string that itself contains the ", " value separator: the row must
// be split into its real columns, not at every comma, so the primary key is
// still the first value and the string survives intact.
func TestMigrateSQLQuotedValueContainingSeparator(t *testing.T) {
	input := "INSERT INTO `widgets` (`id`, `name`, `size`) VALUES\n" +
		"(1, 'Red, green, blue', 3);\n" +
		"\n"

	result, err := MigrateSQL(bufio.NewScanner(strings.NewReader(input)))
	if err != nil {
		t.Fatalf("MigrateSQL returned error: %v", err)
	}

	rowRe := regexp.MustCompile(`\('[0-9a-f-]{36}', 'Red, green, blue', 3\);`)
	if !rowRe.MatchString(result) {
		t.Errorf("expected the row to keep 'Red, green, blue' as one value after a UUID key; got:\n%s", result)
	}
}

// TestMapColumn exercises mapColumn directly, independently of the legacy
// per-table indices in ReplaceValues (which stay unverified, see docs/wip.md).
func TestMapColumn(t *testing.T) {
	idMaps := []IDMap{{TableName: "users", ID: "5", UUID: "user-uuid"}}

	t.Run("in range rewrites the value", func(t *testing.T) {
		values := []string{"'row'", "5", "NULL"}
		if err := mapColumn(values, 1, idMaps, "users", "things"); err != nil {
			t.Fatalf("mapColumn returned error: %v", err)
		}
		if values[1] != "'user-uuid'" {
			t.Errorf("values[1] = %q, want 'user-uuid'", values[1])
		}
		if err := mapColumn(values, 2, idMaps, "users", "things"); err != nil {
			t.Fatalf("mapColumn returned error: %v", err)
		}
		if values[2] != "NULL" {
			t.Errorf("values[2] = %q, want NULL preserved", values[2])
		}
	})

	t.Run("out of range errors", func(t *testing.T) {
		for _, idx := range []int{-1, 3} {
			values := []string{"a", "b", "c"}
			err := mapColumn(values, idx, idMaps, "users", "things")
			if err == nil || !strings.Contains(err.Error(), "things") {
				t.Errorf("mapColumn(idx=%d) error = %v, want one naming the table", idx, err)
			}
		}
	})
}

func TestMigrateDBToV2Panics(t *testing.T) {
	expectPanic := func(t *testing.T) {
		t.Helper()
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected MigrateDBToV2 to panic")
			}
		}()
		MigrateDBToV2()
	}

	t.Run("migration error", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := os.MkdirAll("files", 0755); err != nil {
			t.Fatalf("failed to create files dir: %v", err)
		}
		// Too few columns for the groups mapping, so MigrateSQL errors.
		input := "INSERT INTO `groups` (`id`, `name`) VALUES\n(1, 'Family');\n\n"
		if err := os.WriteFile("files/db.sql", []byte(input), 0644); err != nil {
			t.Fatalf("failed to write db.sql: %v", err)
		}
		expectPanic(t)
		if _, err := os.Stat("files/db_modified_sql_file.sql"); err == nil {
			t.Error("expected no output file after a failed migration")
		}
	})

	t.Run("output not writable", func(t *testing.T) {
		t.Chdir(t.TempDir())
		// A directory in place of the output file makes WriteFile fail.
		if err := os.MkdirAll("files/db_modified_sql_file.sql", 0755); err != nil {
			t.Fatalf("failed to create blocking dir: %v", err)
		}
		input := "INSERT INTO `widgets` (`id`) VALUES\n(1);\n\n"
		if err := os.WriteFile("files/db.sql", []byte(input), 0644); err != nil {
			t.Fatalf("failed to write db.sql: %v", err)
		}
		expectPanic(t)
	})
}
