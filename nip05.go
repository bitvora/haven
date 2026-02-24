package main

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/nbd-wtf/go-nostr/nip05"
)

const MaxNIP05IdentifierLength = 253

// NIP-05 local-part regex
var localPartRegex = regexp.MustCompile("^[a-zA-Z0-9-_.]+$")

func nip05Handler(cfg *nip05.WellKnownResponse) func(w http.ResponseWriter, r *http.Request) {
	if cfg == nil || cfg.Names == nil || len(cfg.Names) == 0 {
		slog.Info("⚠️ NIP-05 handler disabled: no NIP-05 config found with valid identifiers")
		return http.NotFoundHandler().ServeHTTP
	}

	slog.Info("✅ NIP-05 handler enabled")
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract and normalize the 'name' parameter
		name := strings.ToLower(r.URL.Query().Get("name"))

		var response nip05.WellKnownResponse

		// optional: filter response by name
		if name != "" {
			// validate name parameter
			if err := validateName(name); err != nil {
				http.Error(w, "invalid name", http.StatusBadRequest)
				slog.Error("🚫 provided NIP-05 name invalid", "name", name, "error", err)
				return
			}

			pubkey, ok := cfg.Names[name]
			if !ok {
				// return 404 if name not found
				w.WriteHeader(http.StatusNotFound)
				slog.Error("🚫 provided NIP-05 name not found", "name", name)
				return
			}

			// filter Names by name
			response.Names = map[string]string{name: pubkey}
			if cfg.Relays[pubkey] != nil {
				// filter Relays by name
				response.Relays = map[string][]string{
					pubkey: cfg.Relays[pubkey],
				}
			}

			response.NIP46 = cfg.NIP46
		} else {
			// fill response with the entire NIP-05 config
			response = *cfg
		}

		// Set required headers for CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")

		// Encode response
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			slog.Error("🚫 failed to write NIP-05 response", "name", name, "error", err)
			return
		}

		slog.Debug("NIP-05 fetched", "name", name)
	}
}

func getNIP05FromFile(filePath string) *nip05.WellKnownResponse {
	// open the NIP-05 config file (default: nostr.json)
	nip05ConfigFile, err := os.Open(filePath)
	if err != nil {
		slog.Error("🚫 failed to open NIP-05 config file:", "error", err)
		return nil
	}
	defer nip05ConfigFile.Close()

	// Parse and validate NIP-05 config file
	nip05Config, err := parseAndValidateNIP05(nip05ConfigFile)
	if err != nil {
		slog.Error("🚫 failed to parse and validate NIP-05 config file", "error", err)
		return nil
	}

	return nip05Config
}

// parseAndValidateNIP05 reads JSON from an io.Reader and validates NIP-05 data
func parseAndValidateNIP05(r io.Reader) (*nip05.WellKnownResponse, error) {
	var resp nip05.WellKnownResponse

	// Decode the JSON
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to decode JSON: %w", err)
	}

	// Validate 'names' map exists
	if resp.Names == nil || len(resp.Names) == 0 {
		return nil, errors.New("invalid NIP-05: 'names' map is missing or empty")
	}

	// Validate 'names' public keys
	for name, pubkey := range resp.Names {
		if err := validateName(name); err != nil {
			return nil, fmt.Errorf("invalid name '%s': %w", name, err)
		}
		if err := validatePubkey(pubkey); err != nil {
			return nil, fmt.Errorf("invalid public key for name '%s': %w", name, err)
		}
	}

	// Validate 'relays' map (optional)
	if resp.Relays != nil {
		for pubkey, relays := range resp.Relays {
			if err := validatePubkey(pubkey); err != nil {
				return nil, fmt.Errorf("invalid public key in 'relays' mapping: %w", err)
			}
			if err := validateRelayURLs(relays); err != nil {
				return nil, fmt.Errorf("invalid relay URL for pubkey '%s': %w", pubkey, err)
			}
		}
	}

	// Validate 'nip46' map (optional)
	if resp.NIP46 != nil {
		for pubkey, relays := range resp.NIP46 {
			if err := validatePubkey(pubkey); err != nil {
				return nil, fmt.Errorf("invalid public key in 'nip46' mapping: %w", err)
			}
			if err := validateRelayURLs(relays); err != nil {
				return nil, fmt.Errorf("invalid nip46 relay URL for pubkey '%s': %w", pubkey, err)
			}
		}
	}

	return &resp, nil
}

// validateName validates a NIP-05 name
func validateName(name string) error {
	if !localPartRegex.MatchString(name) {
		return errors.New("name contains invalid characters")
	}

	if len(fmt.Sprintf("")) > MaxNIP05IdentifierLength {
		return errors.New("name too long")
	}

	return nil
}

// validatePubkey ensures the pubkey is a 64-character hex string
func validatePubkey(pubkey string) error {
	if len(pubkey) != 64 {
		return fmt.Errorf("must be exactly 64 characters, got %d", len(pubkey))
	}
	_, err := hex.DecodeString(pubkey)
	if err != nil {
		return errors.New("must contain only valid hex characters")
	}
	return nil
}

// validateRelayURLs ensures URLs are valid and use websocket protocols
func validateRelayURLs(urls []string) error {
	for _, rawURL := range urls {
		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("failed to parse URL '%s': %w", rawURL, err)
		}

		scheme := strings.ToLower(parsedURL.Scheme)
		if scheme != "ws" && scheme != "wss" {
			return fmt.Errorf("URL '%s' must start with ws:// or wss://", rawURL)
		}

		if parsedURL.Host == "" {
			return fmt.Errorf("URL '%s' is missing a host", rawURL)
		}
	}
	return nil
}
