package utilities

import (
	"aunefyren/poenskelisten/logger"
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// TableInfo represents information about a table.
type IDMap struct {
	TableName string
	ID        string
	UUID      string
}

// ChangeIDsWithUUIDs replaces the IDs in the SQL data with UUIDs.
func MigrateSQL(sqlContent *bufio.Scanner) (modifiedSQL2 string, err error) {
	modifiedSQL := ""
	modifiedSQL2 = ""
	err = nil
	IDMaps := []IDMap{}
	Tables := []string{}

	// Position variables
	currentMode := "false"
	currentTable := ""

	createTableRegExString := `^CREATE TABLE \x60([\w_]{1,25})\x60 \((\n){0,1}`
	createTableRegEx := regexp.MustCompile(createTableRegExString)
	insertIntoRegExString := `^INSERT INTO \x60([\w_]{1,25})\x60 \([\x60\w, ]{1,}\) VALUES(\n){0,1}`
	insertIntoRegEx := regexp.MustCompile(insertIntoRegExString)
	emptyLineRegExString := `^$`
	emptyLineRegEx := regexp.MustCompile(emptyLineRegExString)
	valueLineRegExString := `^\(.{1,}\)[,;]{1,1}`
	valueLineRegEx := regexp.MustCompile(valueLineRegExString)
	alterTableRegExString := `^ALTER TABLE`
	alterTableRegEx := regexp.MustCompile(alterTableRegExString)

	// Process each table, but only replace ID's
	for sqlContent.Scan() {
		line := sqlContent.Text()
		modifiedLine := sqlContent.Text()

		if createTableRegEx.Match([]byte(line)) {
			currentMode = "create"
			matches := createTableRegEx.FindStringSubmatch(line)
			currentTable = matches[1]
		} else if insertIntoRegEx.Match([]byte(line)) {
			currentMode = "insert"
			matches := insertIntoRegEx.FindStringSubmatch(line)
			currentTable = matches[1]
		} else if currentMode == "insert" && emptyLineRegEx.Match([]byte(line)) {
			currentMode = "none"
			currentTable = "none"
		} else {
			// logger.Log.Info("No Regex matched: " + line)
		}

		if currentMode == "insert" && valueLineRegEx.Match([]byte(line)) {
			logger.Log.Info("INSERT MODE ON TABLE: " + currentTable)
			modifiedLine, IDMaps, err = ReplaceValues(modifiedLine, currentTable, IDMaps, false)
			if err != nil {
				return "", err
			}
		} else if currentMode == "insert" && insertIntoRegEx.Match([]byte(line)) {
			modifiedLine = ChangeColumns(modifiedLine, currentTable)
		} else if currentMode == "create" {
			logger.Log.Info("CREATE MODE ON TABLE: " + currentTable)
			modifiedLine = ChangeColumns(modifiedLine, currentTable)
		}

		if alterTableRegEx.Match([]byte(line)) {
			break
		}

		modifiedSQL += modifiedLine + "\n"
	}

	for _, line := range strings.Split(strings.TrimSuffix(modifiedSQL, "\n"), "\n") {
		modifiedLine := line

		if createTableRegEx.Match([]byte(line)) {
			currentMode = "create"
			matches := createTableRegEx.FindStringSubmatch(line)
			currentTable = matches[1]
		} else if insertIntoRegEx.Match([]byte(line)) {
			currentMode = "insert"
			matches := insertIntoRegEx.FindStringSubmatch(line)
			currentTable = matches[1]
		} else if currentMode == "insert" && emptyLineRegEx.Match([]byte(line)) {
			currentMode = "none"
			currentTable = "none"
		} else {
			// logger.Log.Info("No Regex matched: " + line)
		}

		if currentMode == "insert" && valueLineRegEx.Match([]byte(line)) {
			logger.Log.Info("INSERT MODE ON TABLE: " + currentTable)
			modifiedLine, IDMaps, err = ReplaceValues(modifiedLine, currentTable, IDMaps, true)
			if err != nil {
				return "", err
			}
		} else if currentMode == "create" {
			logger.Log.Info("CREATE MODE ON TABLE: " + currentTable)
			modifiedLine = ChangeColumns(modifiedLine, currentTable)
		}

		if alterTableRegEx.Match([]byte(line)) {
			break
		}

		modifiedSQL2 += modifiedLine + "\n"
	}

	for _, IDMap := range IDMaps {
		alreadyAdded := false
		for _, TableName := range Tables {
			if TableName == IDMap.TableName {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			Tables = append(Tables, IDMap.TableName)
		}
	}

	logger.Log.Info(len(Tables))

	for _, TableName := range Tables {
		modifiedSQL2 += "\n" +
			"ALTER TABLE `" + TableName + "`\n" +
			"	ADD PRIMARY KEY (`id`),\n" +
			"	ADD KEY `idx_" + TableName + "_deleted_at` (`deleted_at`);" +
			"\n"
	}

	modifiedSQL2 += "\nCOMMIT;"

	return
}

func ChangeColumns(line string, currentTable string) (newLine string) {
	newLine = line

	newLine = strings.ReplaceAll(newLine, "bigint(20) UNSIGNED", "varchar(100)")
	newLine = strings.ReplaceAll(newLine, "bigint(20)", "varchar(100)")

	switch currentTable {
	case "groups":
		newLine = strings.ReplaceAll(newLine, "`owner`", "`owner_id`")
	case "group_memberships":
		newLine = strings.ReplaceAll(newLine, "`member`", "`member_id`")
		newLine = strings.ReplaceAll(newLine, "`group`", "`group_id`")
	case "invites":
		newLine = strings.ReplaceAll(newLine, "`invite_code`", "`code`")
		newLine = strings.ReplaceAll(newLine, "`invite_used`", "`used`")
		newLine = strings.ReplaceAll(newLine, "`invite_recipient`", "`recipient_id`")
		newLine = strings.ReplaceAll(newLine, "`invite_enabled`", "`enabled`")
	case "wishes":
		newLine = strings.ReplaceAll(newLine, "`owner`", "`owner_id`")
		newLine = strings.ReplaceAll(newLine, "`wishlist`", "`wishlist_id`")
	case "wishlists":
		newLine = strings.ReplaceAll(newLine, "`owner`", "`owner_id`")
	case "wishlist_collaborators":
		newLine = strings.ReplaceAll(newLine, "`user`", "`user_id`")
		newLine = strings.ReplaceAll(newLine, "`wishlist`", "`wishlist_id`")
	case "wishlist_memberships":
		newLine = strings.ReplaceAll(newLine, "`group`", "`group_id`")
		newLine = strings.ReplaceAll(newLine, "`wishlist`", "`wishlist_id`")
	case "wish_claims":
		newLine = strings.ReplaceAll(newLine, "`wish`", "`wish_id`")
		newLine = strings.ReplaceAll(newLine, "`user`", "`user_id`")
	default:
		logger.Log.Info("No column updates on: " + currentTable)
	}

	return
}

func ReplaceValues(line string, currentTable string, IDMaps []IDMap, secondRun bool) (newLine string, UpdatedIDMaps []IDMap, err error) {
	newLine = line
	UpdatedIDMaps = IDMaps

	startString := "("
	endString := ""

	newLine = strings.TrimPrefix(newLine, "(")
	if strings.HasSuffix(line, "),") {
		newLine = strings.TrimSuffix(newLine, "),")
		endString = "),"
	} else {
		newLine = strings.TrimSuffix(newLine, ");")
		endString = ");"
	}

	values := strings.Split(newLine, ", ")
	if len(values) == 0 {
		logger.Log.Info("Failed to split values for table: " + currentTable)
		return
	}

	finishedLoop := false
	sum := 1
	for sum < 1000 {
		for index, value := range values {
			if strings.HasPrefix(value, "'") && !strings.HasSuffix(value, "'") && index < len(values)+1 {
				newValues := []string{}
				for indexTwo, valueTwo := range values {
					if indexTwo == index+1 {
						newValues[indexTwo-1] += ", " + valueTwo
					} else {
						newValues = append(newValues, valueTwo)
					}

				}
				values = newValues
				break
			}
			if index+1 >= (len(values)) {
				finishedLoop = true
			}
		}
		if finishedLoop {
			break
		}
	}

	// Replace ID
	if !secondRun {
		currentID := values[0]
		newIDMap := IDMap{
			TableName: currentTable,
			ID:        currentID,
			UUID:      uuid.New().String(),
		}
		UpdatedIDMaps = append(UpdatedIDMaps, newIDMap)
		values[0] = "'" + newIDMap.UUID + "'"
	} else {

		// The column indices below encode the exact column order of the old,
		// pre-UUID schema (undocumented anywhere else - see docs/wip.md). If a
		// real dump's column count doesn't match what's assumed here, mapColumn
		// fails loudly instead of letting a stale index either panic or, worse,
		// silently substitute the wrong column's value.
		switch currentTable {
		case "groups":
			err = mapColumn(values, 7, UpdatedIDMaps, "users", currentTable)
		case "group_memberships":
			err = mapColumn(values, 4, UpdatedIDMaps, "groups", currentTable)
			if err == nil {
				err = mapColumn(values, 6, UpdatedIDMaps, "users", currentTable)
			}
		case "invites":
			err = mapColumn(values, 6, UpdatedIDMaps, "users", currentTable)
		case "wishes":
			err = mapColumn(values, 7, UpdatedIDMaps, "users", currentTable)
			if err == nil {
				err = mapColumn(values, 9, UpdatedIDMaps, "wishlists", currentTable)
			}
		case "wishlists":
			err = mapColumn(values, 7, UpdatedIDMaps, "users", currentTable)
		case "wishlist_collaborators":
			err = mapColumn(values, 4, UpdatedIDMaps, "users", currentTable)
			if err == nil {
				err = mapColumn(values, 6, UpdatedIDMaps, "wishlists", currentTable)
			}
		case "wishlist_memberships":
			err = mapColumn(values, 4, UpdatedIDMaps, "groups", currentTable)
			if err == nil {
				err = mapColumn(values, 6, UpdatedIDMaps, "wishlists", currentTable)
			}
		case "wish_claims":
			err = mapColumn(values, 4, UpdatedIDMaps, "wishes", currentTable)
			if err == nil {
				err = mapColumn(values, 5, UpdatedIDMaps, "users", currentTable)
			}
		default:
			logger.Log.Info("No column updates on: " + currentTable)
		}

		if err != nil {
			return "", UpdatedIDMaps, err
		}

	}

	newLineTwo := startString
	for index, value := range values {
		newLineTwo += value
		if index+1 < len(values) {
			newLineTwo += ", "
		}
	}
	newLineTwo += endString

	return newLineTwo, UpdatedIDMaps, nil
}

// mapColumn rewrites values[idx] in place from its legacy numeric ID to the
// migrated UUID for referencedTable. It guards against the hardcoded
// per-table column-index mapping in ReplaceValues no longer matching a real
// dump's column count, which would otherwise panic with an unhelpful
// out-of-range index instead of identifying the actual problem.
func mapColumn(values []string, idx int, idMaps []IDMap, referencedTable string, currentTable string) error {
	if idx < 0 || idx >= len(values) {
		return fmt.Errorf(
			"legacy-schema column mapping for table '%s' expected a column at index %d, but this row only has %d columns - the hardcoded mapping in ReplaceValues no longer matches this dump",
			currentTable, idx, len(values),
		)
	}
	values[idx] = MatchIDToUUID(idMaps, referencedTable, values[idx])
	return nil
}

func MatchIDToUUID(IDMaps []IDMap, currentTable string, ID string) string {
	if ID == "NULL" {
		return "NULL"
	}
	for _, IDMap := range IDMaps {
		if IDMap.TableName == currentTable && IDMap.ID == ID {
			return "'" + IDMap.UUID + "'"
		}
	}
	return "'" + uuid.New().String() + "'"
}

func MigrateDBToV2() {
	// Read SQL file content
	fileContent, err := os.Open("./files/db.sql")
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(fileContent)

	// Call the function to modify the SQL content
	modifiedSQL, err := MigrateSQL(scanner)
	if err != nil {
		panic(err)
	}

	// Write the modified content back to the file
	err = os.WriteFile("./files/db_modified_sql_file.sql", []byte(modifiedSQL), 0644)
	if err != nil {
		panic(err)
	}

	logger.Log.Info("Modification complete. Check './files/db_modified_sql_file.sql'")
}
