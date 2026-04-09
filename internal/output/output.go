package output

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
)

// Table prints a table with a header row and data rows.
func Table(headers []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)
	for _, row := range rows {
		table.Append(row)
	}
	table.Render()
}

// JSON pretty-prints any value as JSON.
func JSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// Success prints a success message.
func Success(msg string) {
	fmt.Println(msg)
}

// Fatal prints an error to stderr and exits.
func Fatal(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}

// Str safely dereferences a *string for display.
func Str(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// IntStr converts an int to string.
func IntStr(i int) string {
	return fmt.Sprintf("%d", i)
}

// BoolStr converts a bool to a display string.
func BoolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
