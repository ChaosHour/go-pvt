package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ViewInfo struct {
	Database  string
	ViewName  string
	CreateSQL string
	AlterSQL  string
	Server    string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <input-file>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	views, err := parseViewsFile(inputFile)
	if err != nil {
		fmt.Printf("Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Create output directory
	outputDir := "flyway-views"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Generate files for each view
	for _, view := range views {
		if err := generateViewFiles(outputDir, view); err != nil {
			fmt.Printf("Error generating files for view %s: %v\n", view.ViewName, err)
			continue
		}
		fmt.Printf("Generated files for %s.%s\n", view.Database, view.ViewName)
	}

	fmt.Printf("Generated %d view file pairs in %s/\n", len(views), outputDir)
}

func parseViewsFile(filename string) ([]ViewInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var views []ViewInfo
	scanner := bufio.NewScanner(file)

	var currentServer, currentDatabase string
	var currentView ViewInfo
	var inCreateSection, inAlterSection bool
	var createSQL, alterSQL strings.Builder
	var createSeparatorSeen, alterSeparatorSeen bool
	var waitForCreateSeparator, waitForAlterSeparator bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Parse server information
		if strings.HasPrefix(line, "Source server:") {
			parts := strings.Split(line, ",")
			if len(parts) >= 2 {
				currentServer = strings.TrimSpace(strings.TrimPrefix(parts[0], "Source server:"))
				dbPart := strings.TrimSpace(parts[1])
				currentDatabase = strings.TrimPrefix(dbPart, "Database: ")
			}
			continue
		}

		// Parse view processing line
		if strings.HasPrefix(line, "Processing view:") {
			// Save previous view if exists
			if currentView.ViewName != "" {
				currentView.CreateSQL = strings.TrimSpace(createSQL.String())
				currentView.AlterSQL = strings.TrimSpace(alterSQL.String())
				fmt.Printf("DEBUG: Parsed view %s - CreateSQL length: %d, AlterSQL length: %d\n",
					currentView.ViewName, len(currentView.CreateSQL), len(currentView.AlterSQL))
				views = append(views, currentView)
			}

			// Start new view
			currentView = ViewInfo{
				Database: currentDatabase,
				ViewName: strings.TrimSpace(strings.TrimPrefix(line, "Processing view:")),
				Server:   currentServer,
			}
			createSQL.Reset()
			alterSQL.Reset()
			inCreateSection = false
			inAlterSection = false
			createSeparatorSeen = false
			alterSeparatorSeen = false
			waitForCreateSeparator = false
			waitForAlterSeparator = false
			fmt.Printf("DEBUG: Starting new view: %s in database: %s\n", currentView.ViewName, currentView.Database)
			continue
		}

		// Parse CREATE statement section
		if line == "Create Statement:" {
			inCreateSection = true
			inAlterSection = false
			waitForCreateSeparator = true // Need to wait for separator before capturing SQL
			fmt.Printf("DEBUG: Entering CREATE section for view: %s\n", currentView.ViewName)
			continue
		}

		// Parse ALTER statement section
		if line == "ALTER VIEW Statement:" {
			inCreateSection = false
			inAlterSection = true
			waitForAlterSeparator = true // Need to wait for separator before capturing SQL
			fmt.Printf("DEBUG: Entering ALTER section for view: %s\n", currentView.ViewName)
			continue
		}

		// Handle separator lines
		if strings.HasPrefix(line, "---") {
			if inCreateSection && waitForCreateSeparator {
				waitForCreateSeparator = false
				createSeparatorSeen = true
				fmt.Printf("DEBUG: First CREATE separator seen, will capture SQL now\n")
				continue
			} else if inCreateSection && createSeparatorSeen {
				inCreateSection = false
				fmt.Printf("DEBUG: Ending CREATE capture for view: %s, captured: %d chars\n",
					currentView.ViewName, createSQL.Len())
				continue
			} else if inAlterSection && waitForAlterSeparator {
				waitForAlterSeparator = false
				alterSeparatorSeen = true
				fmt.Printf("DEBUG: First ALTER separator seen, will capture SQL now\n")
				continue
			} else if inAlterSection && alterSeparatorSeen {
				inAlterSection = false
				fmt.Printf("DEBUG: Ending ALTER capture for view: %s, captured: %d chars\n",
					currentView.ViewName, alterSQL.Len())
				continue
			}
			continue
		}

		// Skip other metadata lines
		if strings.HasPrefix(line, "Connected to") || strings.HasPrefix(line, "Object:") || strings.HasPrefix(line, "Type:") {
			continue
		}

		// Collect CREATE SQL - after first separator but before second
		if inCreateSection && createSeparatorSeen && line != "" {
			if createSQL.Len() > 0 {
				createSQL.WriteString("\n")
			}
			createSQL.WriteString(line)
			fmt.Printf("DEBUG: Capturing CREATE SQL: %s...\n", line[:min(50, len(line))])
		}

		// Collect ALTER SQL - after first separator but before second
		if inAlterSection && alterSeparatorSeen && line != "" {
			if alterSQL.Len() > 0 {
				alterSQL.WriteString("\n")
			}
			alterSQL.WriteString(line)
			fmt.Printf("DEBUG: Capturing ALTER SQL: %s...\n", line[:min(50, len(line))])
		}
	}

	// Don't forget the last view
	if currentView.ViewName != "" {
		currentView.CreateSQL = strings.TrimSpace(createSQL.String())
		currentView.AlterSQL = strings.TrimSpace(alterSQL.String())
		fmt.Printf("DEBUG: Final view %s - CreateSQL length: %d, AlterSQL length: %d\n",
			currentView.ViewName, len(currentView.CreateSQL), len(currentView.AlterSQL))
		views = append(views, currentView)
	}

	return views, scanner.Err()
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func generateViewFiles(outputDir string, view ViewInfo) error {
	// Create database subdirectory
	dbDir := filepath.Join(outputDir, view.Database)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}

	// Generate Ufile.sql (DROP VIEW)
	uFile := filepath.Join(dbDir, fmt.Sprintf("U%s.sql", view.ViewName))
	uContent := formatUFile(view)
	if err := os.WriteFile(uFile, []byte(uContent), 0644); err != nil {
		return err
	}

	// Generate Vfile.sql (CREATE/ALTER VIEW)
	vFile := filepath.Join(dbDir, fmt.Sprintf("V%s.sql", view.ViewName))
	vContent := formatVFile(view)
	if err := os.WriteFile(vFile, []byte(vContent), 0644); err != nil {
		return err
	}

	return nil
}

func formatUFile(view ViewInfo) string {
	// Generate rollback using CREATE OR REPLACE VIEW with ALGORITHM=UNDEFINED
	// Extract the view definition from the ALTER statement
	createReplaceSQL := view.AlterSQL

	// Replace ALTER with CREATE OR REPLACE
	createReplaceSQL = strings.Replace(createReplaceSQL, "ALTER", "CREATE OR REPLACE", 1)

	// Set algorithm to UNDEFINED instead of MERGE
	createReplaceSQL = strings.Replace(createReplaceSQL, "ALGORITHM = MERGE", "ALGORITHM = UNDEFINED", 1)

	formattedSQL := formatSQL(createReplaceSQL)

	return fmt.Sprintf(`-- Flyway Undo Script (Rollback)
-- Database: %s
-- View: %s
-- Changes ALGORITHM back to UNDEFINED using CREATE OR REPLACE

%s
`, view.Database, view.ViewName, formattedSQL)
}

func formatVFile(view ViewInfo) string {
	// Format the ALTER SQL for better readability
	formattedSQL := formatSQL(view.AlterSQL)

	return fmt.Sprintf(`-- Flyway Migration Script
-- Database: %s
-- View: %s

%s
`, view.Database, view.ViewName, formattedSQL)
}

func formatSQL(sql string) string {
	if sql == "" {
		return "-- ERROR: No SQL content found"
	}

	// Remove extra whitespace and format the SQL
	sql = strings.TrimSpace(sql)

	// Add proper line breaks for readability
	sql = regexp.MustCompile(`\s+`).ReplaceAllString(sql, " ")

	// Add line breaks after key SQL keywords
	keywords := []string{
		"ALTER",
		"ALGORITHM",
		"DEFINER",
		"SQL SECURITY",
		"VIEW",
		"AS select",
		"from",
		"join",
		"left join",
		"where",
		"group by",
		"having",
		"order by",
	}

	for _, keyword := range keywords {
		pattern := `(?i)\b` + regexp.QuoteMeta(keyword) + `\b`
		re := regexp.MustCompile(pattern)
		sql = re.ReplaceAllString(sql, "\n    "+keyword)
	}

	// Clean up the formatting
	sql = strings.TrimSpace(sql)
	sql = regexp.MustCompile(`\n\s*\n`).ReplaceAllString(sql, "\n")

	// Remove trailing spaces from each line
	lines := strings.Split(sql, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	sql = strings.Join(lines, "\n")

	// Ensure the statement ends with a semicolon
	if !strings.HasSuffix(sql, ";") {
		sql += ";"
	}

	return sql
}
