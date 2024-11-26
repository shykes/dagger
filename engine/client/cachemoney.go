package client

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/dagger/dagger/engine/slog"

	controlapi "github.com/moby/buildkit/api/services/control"
)

// Load the "cachemoney" experimental config from the environment
func cacheMoneyConfig() (sources, targets []*controlapi.CacheOptionsEntry, err error) {
	sourceNames := os.Getenv("CACHEMONEY_SOURCES")
	for _, name := range strings.Split(sourceNames, ",") {
		if name == "" {
			continue
		}
		source, err := cacheMoneySource(name)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cache money source %q: %w", name, err)
		}
		slog.Info("Loading cachemoney source", "name", name)
		sources = append(sources, source)
	}
	targetNames := os.Getenv("CACHEMONEY_TARGETS")
	for _, name := range strings.Split(targetNames, ",") {
		if name == "" {
			continue
		}
		target, err := cacheMoneyTarget(name)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid cache money target %q: %w", name, err)
		}
		slog.Info("Loading cachemoney target", "name", name)
		targets = append(targets, target)
	}
	return sources, targets, nil
}

// Read a secret from 1password
func opRead(reference string) (string, error) {
	output, err := exec.Command("op", "read", reference).Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute 1password CLI: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func lookupSecret(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok {
		return "", fmt.Errorf("env variable not set: %s", key)
	}
	if strings.HasPrefix(val, "op://") {
		return opRead(val)
	}
	return val, nil
}

func cacheMoneySource(name string) (*controlapi.CacheOptionsEntry, error) {
	// Load standard AWS config
	region := os.Getenv("AWS_DEFAULT_REGION")
	if region == "" {
		region = "us-west-2"
	}
	id, err := lookupSecret("AWS_ACCESS_KEY_ID")
	if err != nil {
		return nil, err
	}
	key, err := lookupSecret("AWS_SECRET_ACCESS_KEY")
	if err != nil {
		return nil, err
	}
	bucket, err := lookupSecret("CACHEMONEY_BUCKET")
	if err != nil {
		return nil, err
	}
	return &controlapi.CacheOptionsEntry{
		Type: "s3",
		Attrs: map[string]string{
			"region":            region,
			"prefix":            "cachemoney/",
			"manifests_prefix":  name + "/",
			"bucket":            bucket,
			"access_key_id":     id,
			"secret_access_key": key,
			"compression":       "zstd",
		},
	}, nil
}

func cacheMoneyTarget(name string) (*controlapi.CacheOptionsEntry, error) {
	source, err := cacheMoneySource(name)
	if err != nil {
		return nil, err
	}
	source.Attrs["mode"] = "max"
	return source, nil
}
