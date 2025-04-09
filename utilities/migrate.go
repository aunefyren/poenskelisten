package utilities

import (
	"aunefyren/poenskelisten/logger"
	"bufio"
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
			modifiedLine, IDMaps = ReplaceValues(modifiedLine, currentTable, IDMaps, false)
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
			modifiedLine, IDMaps = ReplaceValues(modifiedLine, currentTable, IDMaps, true)
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

func ReplaceValues(line string, currentTable string, IDMaps []IDMap, secondRun bool) (newLine string, UpdatedIDMaps []IDMap) {
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

		switch currentTable {
		case "groups":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "users", values[7])
			values[7] = newUUID
		case "group_memberships":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "groups", values[4])
			values[4] = newUUID
			newUUID = MatchIDToUUID(UpdatedIDMaps, "users", values[6])
			values[6] = newUUID
		case "invites":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "users", values[6])
			values[6] = newUUID
		case "wishes":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "users", values[7])
			values[7] = newUUID
			newUUID = MatchIDToUUID(UpdatedIDMaps, "wishlists", values[9])
			values[9] = newUUID
		case "wishlists":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "users", values[7])
			values[7] = newUUID
		case "wishlist_collaborators":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "users", values[4])
			values[4] = newUUID
			newUUID = MatchIDToUUID(UpdatedIDMaps, "wishlists", values[6])
			values[6] = newUUID
		case "wishlist_memberships":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "groups", values[4])
			values[4] = newUUID
			newUUID = MatchIDToUUID(UpdatedIDMaps, "wishlists", values[6])
			values[6] = newUUID
		case "wish_claims":
			newUUID := MatchIDToUUID(UpdatedIDMaps, "wishes", values[4])
			values[4] = newUUID
			newUUID = MatchIDToUUID(UpdatedIDMaps, "users", values[5])
			values[5] = newUUID
		default:
			logger.Log.Info("No column updates on: " + currentTable)
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

	return newLineTwo, UpdatedIDMaps
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
