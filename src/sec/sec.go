package main

import (
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
	"golang.org/x/term"
)

type SecretStore map[string]string

const version = "0.1.1"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version of sec",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("sec version", version)
	},
}

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

func main() {
	var fileFlag string
	var rootCmd = &cobra.Command{
		Use:     "sec",
		Version: "0.0.1",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			key, err := getOrCreateStaticKey()
			if err != nil {
				return err
			}
			store, err := loadStore(expandPath(fileFlag), key)
			if err != nil {
				return err
			}
			// Only prompt for PIN if one is set
			if _, ok := store[pinKey]; ok {
				return requirePIN(store)
			}
			return nil
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

	rootCmd.AddCommand(setCmd, getCmd, deleteCmd, listCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
