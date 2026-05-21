package gateway

import (
	"fmt"
	"testing"
)

func TestDebug_TableOutput(t *testing.T) {
	// Test with special characters in table cells
	input := "| Version | File (path) |\n|---------|-------------|\n| 1.0     | main.go     |\n"
	got := FormatFinalTelegramText(input)
	fmt.Printf("OUTPUT:\n%s\n", got)
	
	// Test with pipe character in cell
	input2 := "| Col A | Col B |\n|-------|-------|\n| a|b   | c|d   |\n"
	got2 := FormatFinalTelegramText(input2)
	fmt.Printf("OUTPUT2:\n%s\n", got2)
	
	// Test with asterisk in cell
	input3 := "| Key | Value |\n|-----|-------|\n| *ptr | test  |\n"
	got3 := FormatFinalTelegramText(input3)
	fmt.Printf("OUTPUT3:\n%s\n", got3)
}
