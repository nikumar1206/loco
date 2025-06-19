package cmd

import "fmt"

// locoOut prints a standard output message with a given prefix.
func locoOut(okPrefix, message string) {
	fmt.Printf("%s%s\n", okPrefix, message)
}

// locoErr prints an error message with a given prefix.
func locoErr(errorPrefix, message string) {
	fmt.Printf("%s%s\n", errorPrefix, message)
}
