package main

import (
	"errors"
	"os"
	"os/user"
	"io/ioutil"
	"bytes"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/openpgp"
)

// Decrypt a password
func Decrypt(filename string, passphrase string) (string, error) {
	var homeDir string
	if usr, err := user.Current(); err == nil {
		homeDir = usr.HomeDir
	}
	secretKeyring := path.Join(homeDir, ".gnupg/secring.gpg")
	// init some vars
	var entity *openpgp.Entity
	var entityList openpgp.EntityList

	// Open the private key file
	keyringFileBuffer, err := os.Open(secretKeyring)
	if err != nil {
		return "", err
	}
	defer keyringFileBuffer.Close()
	entityList, err = openpgp.ReadKeyRing(keyringFileBuffer)
	if err != nil {
		return "", err
	}
	entity = entityList[0]
	// Get the passphrase and read the private key.
	// Have not touched the encrypted string yet
	passphraseByte := []byte(passphrase)
	entity.PrivateKey.Decrypt(passphraseByte)
	for _, subkey := range entity.Subkeys {
		subkey.PrivateKey.Decrypt(passphraseByte)
	}

	// Decrypt it with the contents of the private key
	b, err := ioutil.ReadFile(filename)
	md, err := openpgp.ReadMessage(bytes.NewBuffer(b), entityList, nil, nil)
	if err != nil {
		return "", err
	}
	bytes, err := ioutil.ReadAll(md.UnverifiedBody)
	if err != nil {
		return "", err
	}
	decStr := string(bytes)

	return decStr, nil
}

func match(query string, candidate string) bool {
	lowerQuery := strings.ToLower(query)
	queryParts := strings.Split(lowerQuery, " ")

	lowerCandidate := strings.ToLower(candidate)

	for _, p := range queryParts {
		if !strings.Contains(
			strings.ToLower(lowerCandidate),
			strings.ToLower(p),
		) {
			return false
		}
	}
	return true
}

func query(q string, ps string) []string {
	var passwords []string
	filepath.Walk(ps, func (path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".gpg") {
			passwords = append(passwords, path)
		}
		return nil
	})
	var hits []string
	for _, p := range passwords {
		if match(q, p) {
			hits = append(hits, p)
		}
	}
	return hits
}

func findPasswordshStore() (string, error) {
	var homeDir string
	if usr, err := user.Current(); err == nil {
		homeDir = usr.HomeDir
	}

	pathCandidates := []string {
		os.Getenv("PASSWORD_STORE_DIR"),
		path.Join(homeDir, ".password-store"),
		path.Join(homeDir, "password-store"),
	}

	for _, p := range pathCandidates {
		var err error
		if p, err = filepath.EvalSymlinks(p); err != nil {
			continue
		}
		if _, err = os.Stat(p); err != nil {
			continue
		}
		return p, nil
	}
	return "", errors.New("Couldn't find a valid password store")
}
