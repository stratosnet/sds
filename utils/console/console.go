package console

import (
	"fmt"
	"github.com/stratosnet/sds/utils/cmd"
)

// getPassPhrase retrieves the password associated with an account, either fetched
// from a list of preloaded passphrases, or requested interactively from the user.
func GetPassPhrase(prompt string, confirmation bool, i int) string {
	// Otherwise prompt the user for the password
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		cmd.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			cmd.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			cmd.Fatalf("Passphrases do not match")
		}
	}
	return password
}

// GetAccount return account string from input
func GetAccount(prompt string) string {
	if prompt != "" {
		fmt.Println(prompt)
	}
	account, err := Stdin.PromptInput("Name: ")
	if err != nil {
		cmd.Fatalf("Failed to read account: %v", err)
	}
	if len(account) > 32 {
		account = string([]rune(account)[:32])
	}
	return account
}
