package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type SecretsManager interface {
	GetSecret(ctx context.Context, key string) (string, error)
}

type VaultWardenCliSecretManager struct {
	getPass    func() (string, error)
	cache      map[string]string
	cacheMutex sync.RWMutex
}

var password string

func NewSecretsManager(bwPasFilePath string) (SecretsManager, error) {
	if bwPasFilePath == "" {
		return nil, fmt.Errorf("bwPasFilePath cannot be empty")
	}

	f := func() (string, error) {
		bytes, err := os.ReadFile(bwPasFilePath)
		if err != nil {
			return "", err
		}
		return string(bytes), nil

	}
	return &VaultWardenCliSecretManager{getPass: f, cache: map[string]string{}, cacheMutex: sync.RWMutex{}}, nil
}

func (self *VaultWardenCliSecretManager) GetSecret(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("secret key cannot be empty")
	}

	self.cacheMutex.RLock()
	val, exists := self.cache[key]
	self.cacheMutex.RUnlock()
	if exists {
		return val, nil
	}

	pass, err := self.getPass()
	if err != nil {
		return "", err
	}
	pass = strings.TrimSpace(pass)
	if pass == "" {
		return "", fmt.Errorf("password cannot be empty")
	}
	unlock := exec.CommandContext(ctx, "bw", "unlock", "--passwordenv", "BW_MASTER_PASSWORD", "--raw")

	unlock.Env = append(unlock.Environ(), fmt.Sprintf("BW_MASTER_PASSWORD=%s", pass))

	token, err := unlock.Output()
	if err != nil {
		log.Println("error while getting token: ", err)
		return "", err
	}

	session := strings.TrimSpace(string(token))

	get := exec.CommandContext(ctx, "bw", "get", "password", key, "--raw")
	get.Env = append(get.Environ(), fmt.Sprintf("BW_SESSION=%s", session))

	password, err := get.Output()
	if err != nil {
		return "", err
	}

	val = string(password)
	self.cacheMutex.Lock()
	defer self.cacheMutex.Unlock()

	self.cache[key] = val

	return val, nil

}
