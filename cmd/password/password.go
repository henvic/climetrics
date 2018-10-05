package main

import (
	"fmt"
	"os"
	"syscall"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

func readPassword() (string, error) {
	b, err := terminal.ReadPassword(int(syscall.Stdin))
	return string(b), err
}

func getPassword() (string, error) {
	password, err := readPassword()

	if err != nil {
		return "", err
	}

	var hash []byte
	hash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func main() {
	fmt.Print("Password: ")
	var hash, err = getPassword()
	fmt.Println("█")

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}

	fmt.Println(hash)
}
