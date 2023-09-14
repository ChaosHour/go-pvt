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

// Declare the database connection globally so it can be used by all functions
var db *sql.DB

// Define flags
var (
	source   = flag.String("s", "", "Source Host")
	database = flag.String("d", "", "Database Name")
	show     = flag.Bool("show", false, "Show Databases") // if the -show flag is set, show the databases and exit. Do not try to run the queries.
)

// read the ~/.my.cnf file to get the database credentials
func readMyCnf() error {
	file, err := os.ReadFile(os.Getenv("HOME") + "/.my.cnf")
	if err != nil {
		return err
	}
	lines := strings.Split(string(file), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "user") {
			os.Setenv("MYSQL_USER", strings.TrimSpace(line[5:]))
		}
		if strings.HasPrefix(line, "password") {
			os.Setenv("MYSQL_PASSWORD", strings.TrimSpace(line[9:]))
		}
	}
	return nil
}

// Connect to the database
func connectToDatabase(source, database string) (*sql.DB, error) {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASSWORD"), source, "3306", database))
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

/*
// Get the list of databases
func getDatabases() ([]string, error) {
    var databases []string
    rows, err := db.Query("SHOW DATABASES")
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
*/

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

/*
// Get a list of all procedures, functions, and views using this query an get the database name from the -d flag
func getObjects(database string) ([]string, error) {
    var objects []string
    rows, err := db.Query(fmt.Sprintf("SELECT ROUTINE_NAME, ROUTINE_TYPE, DEFINER FROM INFORMATION_SCHEMA.ROUTINES WHERE ROUTINE_SCHEMA = '%s' UNION ALL SELECT TABLE_NAME, 'VIEW', DEFINER FROM INFORMATION_SCHEMA.VIEWS WHERE TABLE_SCHEMA = '%s' UNION ALL SELECT TRIGGER_NAME, 'TRIGGER', DEFINER FROM INFORMATION_SCHEMA.TRIGGERS WHERE TRIGGER_SCHEMA = '%s' UNION ALL SELECT EVENT_NAME, 'EVENT', DEFINER FROM INFORMATION_SCHEMA.EVENTS WHERE EVENT_SCHEMA = '%s'", database, database, database, database))
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
        objects = append(objects, object+" ("+objectType+") by "+definer)
    }
    return objects, nil
}
*/

// Get a list of all procedures, functions, and views in the specified database
func getObjects(db *sql.DB, database string) ([]string, error) {
	var objects []string
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
		objects = append(objects, object+" ("+objectType+") "+definer)
	}
	return objects, nil
}

// Print the objects
func printResults(objects []string) {
	// Print the total number of objects
	//fmt.Printf("Total: %d\n\n", len(objects))
	// Print the Word "Total:" in green
	fmt.Printf("%s: %d\n", color.YellowString("Total"), len(objects))
	//fmt.Printf("Total: %d\n", len(objects))
	fmt.Println()

	// Print the objects in a MySQL-like table with colors
	fmt.Println(color.GreenString("Objects:"))
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Type", "Definer"})
	/*
		table.SetHeaderColor(tablewriter.Colors{tablewriter.Bold, tablewriter.BgGreenColor},
			tablewriter.Colors{tablewriter.FgHiRedColor, tablewriter.Bold, tablewriter.BgBlackColor},
			tablewriter.Colors{tablewriter.BgCyanColor, tablewriter.FgWhiteColor})
	*/
	table.SetHeaderColor(tablewriter.Colors{tablewriter.Bold, tablewriter.BgGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.BgGreenColor})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("\t") // pad with tabs
	table.SetNoWhiteSpace(true)
	table.SetBorder(true) // enable table border
	for _, object := range objects {
		object := strings.Split(object, " ")
		table.Append(object)
	}

	table.Render()

}

func main() {
	flag.Parse()

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
		// print the Databases: header in green
		fmt.Println(color.GreenString("Databases:"))
		//fmt.Println("Databases:")
		for _, database := range databases {
			fmt.Println(database)
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
