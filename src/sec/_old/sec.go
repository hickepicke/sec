// sec.go
package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/chacha20poly1305"
)

type SecretStore map[string]string

const (
	defaultSecretsFile = "~/.sec.enc"
	pinKey             = "__meta__pin_hash"
)

func expandPath(p string) string {
	if strings.HasPrefix(p, "~") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[1:])
	}
	return p
}

func getOrCreateStaticKey() ([]byte, error) {
	keyPath := expandPath("~/.sec.key")
	if data, err := ioutil.ReadFile(keyPath); err == nil {
		if len(data) == 32 {
			return data, nil
		}
		return nil, fmt.Errorf("invalid key file length")
	} else if os.IsNotExist(err) {
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, err
		}
		if err := ioutil.WriteFile(keyPath, key, 0600); err != nil {
			return nil, err
		}
		return key, nil
	} else {
		return nil, err
	}
}

func encryptStore(store SecretStore, key []byte) ([]byte, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	plaintext, err := json.Marshal(store)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ciphertext := aead.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func decryptStore(data []byte, key []byte) (SecretStore, error) {
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}
	if len(data) < chacha20poly1305.NonceSizeX {
		return nil, errors.New("invalid ciphertext: too short")
	}
	nonce := data[:chacha20poly1305.NonceSizeX]
	ciphertext := data[chacha20poly1305.NonceSizeX:]
	plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	var store SecretStore
	if err := json.Unmarshal(plaintext, &store); err != nil {
		return nil, err
	}
	return store, nil
}

func loadStore(path string, key []byte) (SecretStore, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(SecretStore), nil
		}
		return nil, err
	}
	return decryptStore(data, key)
}

func saveStore(path string, store SecretStore, key []byte) error {
	data, err := encryptStore(store, key)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0600)
}

func exportPlaintext(store SecretStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func importPlaintext(path string) (SecretStore, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var store SecretStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	return store, nil
}

func promptPIN() (string, error) {
	fmt.Print("Enter PIN: ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	return "", scanner.Err()
}

func requirePIN(store SecretStore) error {
	hash, ok := store[pinKey]
	if !ok {
		return nil // no PIN set
	}
	pin, err := promptPIN()
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin)); err != nil {
		return errors.New("incorrect PIN")
	}
	return nil
}

func main() {
	var fileFlag string
	var rootCmd = &cobra.Command{
		Use: "sec",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if fileFlag == "" {
				fileFlag = defaultSecretsFile
			}
			key, err := getOrCreateStaticKey()
			if err != nil {
				return err
			}
			store, err := loadStore(expandPath(fileFlag), key)
			if err != nil {
				return err
			}
			return requirePIN(store)
		},
	}
	rootCmd.PersistentFlags().StringVarP(&fileFlag, "file", "f", defaultSecretsFile, "Path to secrets file")

	var setCmd = &cobra.Command{
		Use:   "set",
		Short: "Set a secret value",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			key, err := getOrCreateStaticKey()
			if err != nil {
				log.Fatal(err)
			}
			store, err := loadStore(expandPath(fileFlag), key)
			if err != nil {
				log.Fatal(err)
			}
			store[args[0]] = args[1]
			if err := saveStore(expandPath(fileFlag), store, key); err != nil {
				log.Fatal(err)
			}
			fmt.Println("Secret set.")
		},
	}

	var getCmd = &cobra.Command{
		Use:   "get",
		Short: "Get a secret value",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			key, err := getOrCreateStaticKey()
			if err != nil {
				log.Fatal(err)
			}
			store, err := loadStore(expandPath(fileFlag), key)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Println(store[args[0]])
		},
	}

	var deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			key, err := getOrCreateStaticKey()
			if err != nil {
				log.Fatal(err)
			}
			store, err := loadStore(expandPath(fileFlag), key)
			if err != nil {
				log.Fatal(err)
			}
			delete(store, args[0])
			if err := saveStore(expandPath(fileFlag), store, key); err != nil {
				log.Fatal(err)
			}
			fmt.Println("Secret deleted.")
		},
	}

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all stored keys",
		Run: func(cmd *cobra.Command, args []string) {
			key, err := getOrCreateStaticKey()
			if err != nil {
				log.Fatal(err)
			}
			store, err := loadStore(expandPath(fileFlag), key)
			if err != nil {
				log.Fatal(err)
			}
			for k := range store {
				if k != pinKey {
					fmt.Println(k)
				}
			}
		},
	}

	var setPinCmd = &cobra.Command{
		Use:   "set-pin",
		Short: "Set a new PIN to lock the vault",
		Run: func(cmd *cobra.Command, args []string) {
			key, err := getOrCreateStaticKey()
			if err != nil {
				log.Fatalf("Failed to get key: %v", err)
			}
			store, err := loadStore(expandPath(fileFlag), key)
			if err != nil {
				log.Fatalf("Failed to load store: %v", err)
			}
			if _, exists := store[pinKey]; exists {
				fmt.Println("PIN already set. Use change-pin or remove-pin instead.")
				return
			}
			fmt.Print("Enter new PIN: ")
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() {
				log.Fatalf("No input")
			}
			pin := scanner.Text()
			hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("Failed to hash PIN: %v", err)
			}
			store[pinKey] = string(hash)
			if err := saveStore(expandPath(fileFlag), store, key); err != nil {
				log.Fatalf("Failed to save store: %v", err)
			}
			fmt.Println("PIN set.")
		},
	}

	var changePinCmd = &cobra.Command{
		Use:   "change-pin",
		Short: "Change the current PIN",
		Run: func(cmd *cobra.Command, args []string) {
			key, err := getOrCreateStaticKey()
			if err != nil {
				log.Fatalf("Failed to get key: %v", err)
			}
			store, err := loadStore(expandPath(fileFlag), key)
			if err != nil {
				log.Fatalf("Failed to load store: %v", err)
			}
			if err := requirePIN(store); err != nil {
				log.Fatalf("Authentication failed: %v", err)
			}
			fmt.Print("Enter new PIN: ")
			scanner := bufio.NewScanner(os.Stdin)
			if !scanner.Scan() {
				log.Fatalf("No input")
			}
			pin := scanner.Text()
			hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
			if err != nil {
				log.Fatalf("Failed to hash PIN: %v", err)
			}
			store[pinKey] = string(hash)
			if err := saveStore(expandPath(fileFlag), store, key); err != nil {
				log.Fatalf("Failed to save store: %v", err)
			}
			fmt.Println("PIN changed.")
		},
	}

	var removePinCmd = &cobra.Command{
		Use:   "remove-pin",
		Short: "Remove the current PIN",
		Run: func(cmd *cobra.Command, args []string) {
			key, err := getOrCreateStaticKey()
			if err != nil {
				log.Fatalf("Failed to get key: %v", err)
			}
			store, err := loadStore(expandPath(fileFlag), key)
			if err != nil {
				log.Fatalf("Failed to load store: %v", err)
			}
			if err := requirePIN(store); err != nil {
				log.Fatalf("Authentication failed: %v", err)
			}
			delete(store, pinKey)
			if err := saveStore(expandPath(fileFlag), store, key); err != nil {
				log.Fatalf("Failed to save store: %v", err)
			}
			fmt.Println("PIN removed.")
		},
	}

	rootCmd.AddCommand(setCmd, getCmd, deleteCmd, listCmd)
	rootCmd.AddCommand(setPinCmd, changePinCmd, removePinCmd)
	rootCmd.Execute()
}
