package main

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

const SecretStoreFile = "~/.sec.enc"

func setPIN(store SecretStore) error {
	pin, err := promptPIN("Enter new PIN: ")
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	store[pinKey] = string(hash)
	return nil
}

func promptPIN(prompt string) (string, error) {
	fmt.Print(prompt)
	bytePIN, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(bytePIN), nil
}

func requirePIN(store SecretStore) error {
	hash, ok := store[pinKey]
	if !ok {
		return nil // No PIN set
	}

	const maxAttempts = 3
	for i := 0; i < maxAttempts; i++ {
		pin, err := promptPIN("Enter PIN: ")
		if err != nil {
			return err
		}
		if bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin)) == nil {
			return nil
		}
		fmt.Println("Incorrect PIN.")
	}
	return errors.New("too many incorrect attempts")
}
