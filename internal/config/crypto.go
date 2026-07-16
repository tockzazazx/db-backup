package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// encPrefix marks values produced by EncryptSecret in the config file.
const encPrefix = "enc1:"

// machineID returns a stable per-machine identifier used to derive the
// encryption key, so an encrypted config value is useless when the file is
// copied to another machine.
func machineID() (string, error) {
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if b, err := os.ReadFile(p); err == nil {
			if id := strings.TrimSpace(string(b)); id != "" {
				return id, nil
			}
		}
	}
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "IOPlatformUUID") {
					if parts := strings.Split(line, "\""); len(parts) >= 4 {
						return parts[3], nil
					}
				}
			}
		}
	}
	return "", errors.New("cannot determine a machine id to derive the secret encryption key")
}

func secretCipher() (cipher.AEAD, error) {
	id, err := machineID()
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256([]byte("boxdb-secret-v1:" + id))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

// EncryptSecret seals a secret with AES-256-GCM under a machine-bound key.
func EncryptSecret(plain string) (string, error) {
	gcm, err := secretCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptSecret opens a value produced by EncryptSecret on this same machine.
func DecryptSecret(stored string) (string, error) {
	if !strings.HasPrefix(stored, encPrefix) {
		return "", fmt.Errorf("value is not an encrypted secret (missing %q prefix)", encPrefix)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, encPrefix))
	if err != nil {
		return "", fmt.Errorf("corrupt encrypted secret: %w", err)
	}
	gcm, err := secretCipher()
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("corrupt encrypted secret: too short")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("cannot decrypt secret — it was most likely encrypted on a different machine; set it again on this one")
	}
	return string(plain), nil
}
