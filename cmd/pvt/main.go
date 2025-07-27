// A Go cli that will find all procedures, functions, and views with definer.

package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	_ "github.com/go-sql-driver/mysql"
	"github.com/olekukonko/tablewriter"
)

// Define flags
var (
	source     = flag.String("s", "", "Source Host")
	database   = flag.String("d", "", "Database Name")
	show       = flag.Bool("show", false, "Show Databases") // if the -show flag is set, show the databases and exit. Do not try to run the queries.
	showCreate = flag.String("show-create", "", "Show CREATE statement for specified object name")
	algorithm  = flag.String("algo", "", "Set algorithm for view (MERGE, TEMPTABLE)")
	execute    = flag.Bool("true", false, "Execute the ALTER VIEW statement when -algo is specified")
)

// read the ~/.my.cnf file to get the database credentials
func readMyCnf() error {
	file, err := os.ReadFile(os.Getenv("HOME") + "/.my.cnf")
	if err != nil {
		return err
	}
	lines := strings.Split(string(file), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "user") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				os.Setenv("MYSQL_USER", strings.TrimSpace(parts[1]))
			}
		}
		if strings.HasPrefix(line, "password") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				os.Setenv("MYSQL_PASSWORD", strings.TrimSpace(parts[1]))
			}
		}
	}
	return nil
}

// Connect to the database
func connectToDatabase(source, database string) (*sql.DB, error) {
	// Use information_schema if no specific database is provided
	dbName := database
	if dbName == "" {
		dbName = "information_schema"
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASSWORD"), source, "3306", dbName))
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}

	// Get the hostname of the connected MySQL server
	var hostname string
	err = db.QueryRow("SELECT @@hostname").Scan(&hostname)
	if err != nil {
		return nil, err
	}

	// Print the result
	fmt.Printf("Connected to %s (%s): %s\n", source, hostname, color.GreenString("✔"))
	fmt.Println()

	return db, nil
}

// Get the list of databases
func getDatabases(db *sql.DB) ([]string, error) {
	var databases []string
	rows, err := db.Query("SELECT schema_name FROM information_schema.schemata")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var database string
		err := rows.Scan(&database)
		if err != nil {
			return nil, err
		}
		databases = append(databases, database)
	}
	return databases, nil
}

// Get a list of all procedures, functions, and views in the specified database
func getObjects(db *sql.DB, database string) ([][]string, error) {
	var objects [][]string
	query := `
        SELECT ROUTINE_NAME, ROUTINE_TYPE, DEFINER FROM INFORMATION_SCHEMA.ROUTINES WHERE ROUTINE_SCHEMA = ?
        UNION ALL
        SELECT TABLE_NAME, 'VIEW', DEFINER FROM INFORMATION_SCHEMA.VIEWS WHERE TABLE_SCHEMA = ?
        UNION ALL
        SELECT TRIGGER_NAME, 'TRIGGER', DEFINER FROM INFORMATION_SCHEMA.TRIGGERS WHERE TRIGGER_SCHEMA = ?
        UNION ALL
        SELECT EVENT_NAME, 'EVENT', DEFINER FROM INFORMATION_SCHEMA.EVENTS WHERE EVENT_SCHEMA = ?
    `
	rows, err := db.Query(query, database, database, database, database)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var object, objectType, definer string
		err := rows.Scan(&object, &objectType, &definer)
		if err != nil {
			return nil, err
		}
		objects = append(objects, []string{object, objectType, definer})
	}
	return objects, nil
}

// Get CREATE statement for a specific object
func getCreateStatement(db *sql.DB, database, objectName string) error {
	// First, check if the object exists and determine its type
	var objectType string
	typeQuery := `
		SELECT 'VIEW' as type FROM INFORMATION_SCHEMA.VIEWS 
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		UNION ALL
		SELECT ROUTINE_TYPE as type FROM INFORMATION_SCHEMA.ROUTINES 
		WHERE ROUTINE_SCHEMA = ? AND ROUTINE_NAME = ?
		UNION ALL
		SELECT 'TRIGGER' as type FROM INFORMATION_SCHEMA.TRIGGERS 
		WHERE TRIGGER_SCHEMA = ? AND TRIGGER_NAME = ?
		UNION ALL
		SELECT 'EVENT' as type FROM INFORMATION_SCHEMA.EVENTS 
		WHERE EVENT_SCHEMA = ? AND EVENT_NAME = ?
		LIMIT 1
	`

	err := db.QueryRow(typeQuery, database, objectName, database, objectName, database, objectName, database, objectName).Scan(&objectType)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("object '%s' not found in database '%s'", objectName, database)
		}
		return fmt.Errorf("error determining object type: %v", err)
	}

	// Now that we know the object type, we can get its create statement
	var query string
	switch objectType {
	case "VIEW":
		// For views, we need a different approach
		var viewName, createStatement, characterSetClient, collationConnection string
		query = fmt.Sprintf("SHOW CREATE VIEW `%s`.`%s`", database, objectName)
		err := db.QueryRow(query).Scan(&viewName, &createStatement, &characterSetClient, &collationConnection)
		if err != nil {
			return fmt.Errorf("error getting CREATE VIEW statement: %v", err)
		}
		printCreateStatement(viewName, "VIEW", createStatement)
		return nil
	case "PROCEDURE", "FUNCTION":
		query = fmt.Sprintf("SHOW CREATE %s `%s`.`%s`", objectType, database, objectName)
		var name, sqlMode, createStatement, charset, collation, dbCollation string
		err := db.QueryRow(query).Scan(&name, &sqlMode, &createStatement, &charset, &collation, &dbCollation)
		if err != nil {
			return fmt.Errorf("error getting CREATE %s statement: %v", objectType, err)
		}
		printCreateStatement(name, objectType, createStatement)
		return nil
	case "TRIGGER":
		query = fmt.Sprintf("SHOW CREATE TRIGGER `%s`.`%s`", database, objectName)
		var triggerName, sqlMode, createStatement, charset, collation, dbCollation string
		err := db.QueryRow(query).Scan(&triggerName, &sqlMode, &createStatement, &charset, &collation, &dbCollation)
		if err != nil {
			return fmt.Errorf("error getting CREATE TRIGGER statement: %v", err)
		}
		printCreateStatement(triggerName, "TRIGGER", createStatement)
		return nil
	case "TABLE":
		query = fmt.Sprintf("SHOW CREATE TABLE `%s`.`%s`", database, objectName)
		var tableName, createStatement string
		err := db.QueryRow(query).Scan(&tableName, &createStatement)
		if err != nil {
			return fmt.Errorf("error getting CREATE TABLE statement: %v", err)
		}
		printCreateStatement(tableName, "TABLE", createStatement)
		return nil
	case "EVENT":
		query = fmt.Sprintf("SHOW CREATE EVENT `%s`.`%s`", database, objectName)
		var eventName, createStatement string
		err := db.QueryRow(query).Scan(&eventName, &createStatement)
		if err != nil {
			return fmt.Errorf("error getting CREATE EVENT statement: %v", err)
		}
		printCreateStatement(eventName, "EVENT", createStatement)
		return nil
	default:
		return fmt.Errorf("unknown object type '%s' for '%s'", objectType, objectName)
	}
}

// Print CREATE statement vertically
func printCreateStatement(name, objType, createStatement string) {
	fmt.Printf("%s: %s\n", color.YellowString("Object"), name)
	fmt.Printf("%s: %s\n", color.YellowString("Type"), objType)
	fmt.Printf("%s:\n", color.GreenString("Create Statement"))
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println(createStatement)
	fmt.Println(strings.Repeat("-", 80))
}

// Print the objects
func printResults(objects [][]string) {
	// Print the total number of objects
	fmt.Printf("%s: %d\n", color.YellowString("Total"), len(objects))
	fmt.Println()

	// Print the objects in a MySQL-like table with colors
	fmt.Println(color.New(color.FgGreen).Sprint("Objects:"))
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Type", "Definer"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding(" ")
	table.SetNoWhiteSpace(false)
	table.SetBorder(false)

	for _, object := range objects {
		table.Append(object)
	}

	table.Render()
}

// Generate ALTER VIEW statement with specified algorithm
func generateAlterViewStatement(db *sql.DB, database, viewName, algorithm string) (string, error) {
	// Get the current view definition
	var viewDef, createStmt, characterSetClient, collationConnection string
	query := fmt.Sprintf("SHOW CREATE VIEW `%s`.`%s`", database, viewName)

	err := db.QueryRow(query).Scan(&viewDef, &createStmt, &characterSetClient, &collationConnection)
	if err != nil {
		return "", fmt.Errorf("error getting view definition: %v", err)
	}

	// Extract the current definer and security settings
	definerStart := strings.Index(createStmt, "DEFINER=")
	definerEnd := strings.Index(createStmt[definerStart:], " SQL SECURITY")
	if definerStart == -1 || definerEnd == -1 {
		return "", fmt.Errorf("could not parse view definition")
	}

	definer := createStmt[definerStart : definerStart+definerEnd]

	// Extract SQL SECURITY setting
	securityStart := strings.Index(createStmt, "SQL SECURITY")
	securityEnd := strings.Index(createStmt[securityStart:], " VIEW")
	if securityStart == -1 || securityEnd == -1 {
		return "", fmt.Errorf("could not parse SQL SECURITY setting")
	}

	security := createStmt[securityStart : securityStart+securityEnd]

	// Extract the view definition (after AS)
	asStart := strings.Index(createStmt, " AS ")
	if asStart == -1 {
		return "", fmt.Errorf("could not find view definition")
	}

	viewDefinition := createStmt[asStart+4:]

	// Build the ALTER VIEW statement with semicolon at the end
	alterStmt := fmt.Sprintf("ALTER \n    ALGORITHM = %s\n    %s\n    %s\n    VIEW `%s`.`%s` AS %s;",
		algorithm, definer, security, database, viewName, viewDefinition)

	return alterStmt, nil
}

// Execute ALTER VIEW statement
func executeAlterViewStatement(db *sql.DB, alterStmt string) error {
	_, err := db.Exec(alterStmt)
	return err
}

// Handle view algorithm settings
func handleViewAlgorithm(db *sql.DB, database, viewName, algorithm string, execute bool) error {
	// Validate algorithm
	algorithm = strings.ToUpper(algorithm)
	if algorithm != "MERGE" && algorithm != "TEMPTABLE" && algorithm != "UNDEFINED" {
		return fmt.Errorf("invalid algorithm: %s. Must be MERGE, TEMPTABLE, or UNDEFINED", algorithm)
	}

	// Generate the ALTER VIEW statement
	alterStmt, err := generateAlterViewStatement(db, database, viewName, algorithm)
	if err != nil {
		return err
	}

	// Print the ALTER VIEW statement
	fmt.Printf("%s:\n", color.GreenString("ALTER VIEW Statement"))
	fmt.Println(strings.Repeat("-", 80))
	fmt.Println(alterStmt)
	fmt.Println(strings.Repeat("-", 80))

	// If execute flag is set, execute the ALTER VIEW statement
	if execute {
		fmt.Printf("Executing ALTER VIEW statement... ")
		err := executeAlterViewStatement(db, alterStmt)
		if err != nil {
			fmt.Printf("%s\n", color.RedString("Failed"))
			return fmt.Errorf("error executing ALTER VIEW statement: %v", err)
		}
		fmt.Printf("%s\n", color.GreenString("Success"))
	}

	return nil
}

func main() {
	flag.Parse()

	// if no flags are set, print the usage and exit
	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(0)
	}

	// make sure that source and at least one of database or show is set
	if *source == "" || (*database == "" && !*show && *showCreate == "") {
		flag.Usage()
		os.Exit(1)
	}

	// make sure database is provided when using show-create
	if *showCreate != "" && *database == "" {
		fmt.Println("Error: -d (database) flag is required when using -show-create")
		flag.Usage()
		os.Exit(1)
	}

	// Read the ~/.my.cnf file to get the database credentials
	err := readMyCnf()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Connect to the database
	db, err := connectToDatabase(*source, *database)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()

	// Get the list of databases
	databases, err := getDatabases(db)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// If the -show flag is set, show the databases and exit. Do not try to run the queries.
	if *show {
		fmt.Println(color.GreenString("Databases:"))
		for _, database := range databases {
			fmt.Println(database)
		}
		os.Exit(0)
	}

	// If the -show-create flag is set, show CREATE statement and exit
	if *showCreate != "" {
		// First show the CREATE statement
		err := getCreateStatement(db, *database, *showCreate)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// If algorithm is specified and it's a view, handle algorithm change
		if *algorithm != "" {
			// Check if the object is a view
			var count int
			err := db.QueryRow("SELECT COUNT(*) FROM information_schema.views WHERE table_schema = ? AND table_name = ?",
				*database, *showCreate).Scan(&count)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			if count > 0 {
				// It's a view, handle algorithm change
				err := handleViewAlgorithm(db, *database, *showCreate, *algorithm, *execute)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
			} else {
				fmt.Printf("Error: -algo flag can only be used with views. %s is not a view.\n", *showCreate)
				os.Exit(1)
			}
		}

		os.Exit(0)
	}

	// Get a list of all procedures, functions, and views in the specified database
	objects, err := getObjects(db, *database)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Print the objects in a MySQL-like table with colors
	printResults(objects)
}
