package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"

	"github.com/rennaisance-jomt/axon/internal/storage"
)

var (
	ErrSecretNotFound = errors.New("secret not found")
	ErrDomainMismatch = errors.New("secret domain mismatch (phishing protection triggered)")
)

// Secret represents a stored credential or sensitive value.
// It is bound to a specific Domain to prevent cross-origin exfiltration.
type Secret struct {
	Name     string `json:"name"`
	Domain   string `json:"domain"` // Bind the secret to this domain (e.g. github.com)
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Value    string `json:"value,omitempty"` // General purpose secret
	CreatedAt time.Time `json:"created_at"`
	Labels    []string `json:"labels,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Vault manages secure credentials using an underlying BadgerDB storage.
// All secrets are encrypted at rest using AES-256-GCM.
type Vault struct {
	db        *storage.DB
	masterKey []byte
}

// NewVault creates a new vault instance with the provided storage and master key.
func NewVault(db *storage.DB, masterKey []byte) *Vault {
	return &Vault{
		db:        db,
		masterKey: masterKey,
	}
}

// AddSecret stores a new secret in the vault
func (v *Vault) AddSecret(secret *Secret) error {
	// Normalize domain to base domain for wider matching (TLD+1)
	secret.Domain = GetBaseDomain(secret.Domain)

	data, err := json.Marshal(secret)
	if err != nil {
		return err
	}

	encrypted, err := v.encrypt(data)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("vault:secret:%s:%s", secret.Domain, secret.Name)
	return v.db.SetSession(key, encrypted) 
}

// GetSecret retrieves a secret by name and validates its domain binding.
// If the currentDomainOrURL does not match the secret's bound domain (using TLD+1),
// it returns ErrDomainMismatch to prevent phishing attacks.
func (v *Vault) GetSecret(name, currentDomainOrURL string) (*Secret, error) {
	currentDomain := GetBaseDomain(currentDomainOrURL)
	
	// We check for exact match or the base domain match
	key := fmt.Sprintf("vault:secret:%s:%s", currentDomain, name)
	encrypted, err := v.db.GetSession(key)
	if err != nil {
		return nil, ErrSecretNotFound
	}

	data, err := v.decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	var secret Secret
	if err := json.Unmarshal(data, &secret); err != nil {
		return nil, err
	}

	// Double check domain binding (security layer)
	// Must be same base domain
	if GetBaseDomain(currentDomain) != GetBaseDomain(secret.Domain) {
		return nil, ErrDomainMismatch
	}

	return &secret, nil
}

// ListSecretsByDomain returns all secrets matching a domain (supports TLD+1)
func (v *Vault) ListSecretsByDomain(currentDomainOrURL string) ([]Secret, error) {
	domain := GetBaseDomain(currentDomainOrURL)
	prefix := fmt.Sprintf("session:vault:secret:%s:", domain)
	
	// We need a way to list from DB. For now, let's assume we can use the storage wrapper.
	// Since storage.DB uses "session:" prefix for SetSession, we include it.
	
	keys, err := v.db.ListWithPrefix(prefix)
	if err != nil {
		return nil, err
	}

	var secrets []Secret
	for _, key := range keys {
		// key is returned without "session:" prefix from ListWithPrefix if we implement it that way
		encrypted, err := v.db.GetSession(key)
		if err != nil {
			continue
		}

		data, err := v.decrypt(encrypted)
		if err != nil {
			continue
		}

		var secret Secret
		if err := json.Unmarshal(data, &secret); err == nil {
			secrets = append(secrets, secret)
		}
	}

	return secrets, nil
}

// GetBaseDomain extracts TLD+1 from a URL or domain string
func GetBaseDomain(input string) string {
	input = strings.TrimSpace(strings.ToLower(input))
	if strings.Contains(input, "://") {
		u, err := url.Parse(input)
		if err == nil {
			input = u.Hostname()
		}
	}

	if !strings.Contains(input, ".") {
		return input
	}

	eTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(input)
	if err == nil {
		return eTLDPlusOne
	}

	// Fallback to exactly what we were given if it can't be parsed
	return input
}

// encrypt encrypts data using AES-GCM
func (v *Vault) encrypt(data []byte) ([]byte, error) {
	if len(v.masterKey) == 0 {
		return data, nil // No encryption if key is missing (fallback/dev mode)
	}

	block, err := aes.NewCipher(v.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

// decrypt decrypts data using AES-GCM
func (v *Vault) decrypt(data []byte) ([]byte, error) {
	if len(v.masterKey) == 0 {
		return data, nil
	}

	block, err := aes.NewCipher(v.masterKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
