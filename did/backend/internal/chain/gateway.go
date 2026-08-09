package chain

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"didcontract/backend/internal/config"
	"didcontract/backend/internal/domain"
	"didcontract/sdk-go/client"
	"gopkg.in/yaml.v3"
)

type Status struct {
	Chain    client.ChainMetadata `json:"chain"`
	Contract struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Runtime string `json:"runtime"`
		Status  string `json:"status"`
	} `json:"contract"`
	Actors []domain.Actor `json:"actors"`
}

type Gateway interface {
	Call(ctx context.Context, actorAlias, method string, request, out interface{}) (*client.Response, error)
	Actor(alias string) (domain.Actor, error)
	Origin(alias string) (string, error)
	Ready(ctx context.Context) error
	Status(ctx context.Context) (Status, error)
	Close()
}

type actorClient struct {
	actor    domain.Actor
	chain    *client.ChainClient
	contract *client.Contract
}

type Pool struct {
	mu      sync.RWMutex
	clients map[string]*actorClient
}

func NewPool(cfg config.Config) (*Pool, error) {
	pool := &Pool{clients: make(map[string]*actorClient, len(cfg.Actors))}
	for _, actorConfig := range cfg.Actors {
		preparedConfig, cleanup, err := prepareSDKConfig(actorConfig.SDKConfigPath, actorConfig.SDKWorkingDir)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("prepare actor %s SDK configuration: %w", actorConfig.Alias, err)
		}
		chainClient, err := client.New(client.Config{
			SDKConfigPath: preparedConfig,
			ContractName:  cfg.Chain.ContractName,
			Timeout:       cfg.Chain.Timeout,
			DisableSDKLog: true,
		})
		cleanup()
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("initialize actor %s: %w", actorConfig.Alias, err)
		}
		pool.clients[actorConfig.Alias] = &actorClient{
			actor: domain.Actor{Alias: actorConfig.Alias, DID: actorConfig.DID},
			chain: chainClient, contract: client.NewContract(chainClient),
		}
	}
	return pool, nil
}

func prepareSDKConfig(configPath, workingDirectory string) (string, func(), error) {
	if strings.TrimSpace(workingDirectory) == "" {
		return configPath, func() {}, nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", func() {}, err
	}
	var document interface{}
	if err := yaml.Unmarshal(data, &document); err != nil {
		return "", func() {}, err
	}
	document = absolutizeSDKPaths(document, workingDirectory, "")
	encoded, err := yaml.Marshal(document)
	if err != nil {
		return "", func() {}, err
	}
	temporary, err := os.CreateTemp(filepath.Dir(configPath), ".backend-sdk-*.yml")
	if err != nil {
		return "", func() {}, err
	}
	name := temporary.Name()
	cleanup := func() { _ = os.Remove(name) }
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		cleanup()
		return "", func() {}, err
	}
	if _, err := temporary.Write(encoded); err != nil {
		temporary.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := temporary.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return name, cleanup, nil
}

func absolutizeSDKPaths(value interface{}, base, key string) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		for childKey, child := range typed {
			typed[childKey] = absolutizeSDKPaths(child, base, childKey)
		}
		return typed
	case []interface{}:
		for index, child := range typed {
			typed[index] = absolutizeSDKPaths(child, base, key)
		}
		return typed
	case string:
		if (strings.HasSuffix(key, "_file_path") || key == "trust_root_paths") && typed != "" && !filepath.IsAbs(typed) {
			return filepath.Clean(filepath.Join(base, typed))
		}
		return typed
	default:
		return value
	}
}

func (p *Pool) get(alias string) (*actorClient, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	value, ok := p.clients[alias]
	if !ok {
		return nil, fmt.Errorf("unknown actor %s", alias)
	}
	return value, nil
}

func (p *Pool) Call(ctx context.Context, actorAlias, method string, request, out interface{}) (*client.Response, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	actor, err := p.get(actorAlias)
	if err != nil {
		return nil, err
	}
	return actor.contract.Call(method, request, out)
}

func (p *Pool) Actor(alias string) (domain.Actor, error) {
	value, err := p.get(alias)
	if err != nil {
		return domain.Actor{}, err
	}
	if _, err := p.Origin(alias); err != nil {
		return domain.Actor{}, err
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return value.actor, nil
}

func (p *Pool) Origin(alias string) (string, error) {
	value, err := p.get(alias)
	if err != nil {
		return "", err
	}
	p.mu.RLock()
	origin := value.actor.Origin
	p.mu.RUnlock()
	if origin != "" {
		return origin, nil
	}
	origin, err = value.chain.OriginAddress()
	if err != nil {
		return "", fmt.Errorf("derive origin for actor %s: %w", alias, err)
	}
	p.mu.Lock()
	value.actor.Origin = origin
	p.mu.Unlock()
	return origin, nil
}

func (p *Pool) Ready(ctx context.Context) error {
	p.mu.RLock()
	clients := make([]*actorClient, 0, len(p.clients))
	for _, value := range p.clients {
		clients = append(clients, value)
	}
	p.mu.RUnlock()
	if len(clients) == 0 {
		return errors.New("actor client pool is empty")
	}
	var chainID string
	for _, value := range clients {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, err := p.Origin(value.actor.Alias); err != nil {
			return err
		}
		if err := value.chain.Ping(); err != nil {
			return fmt.Errorf("actor %s chain ping: %w", value.actor.Alias, err)
		}
		metadata, err := value.chain.Metadata()
		if err != nil {
			return fmt.Errorf("actor %s metadata: %w", value.actor.Alias, err)
		}
		if chainID == "" {
			chainID = metadata.ChainID
		} else if metadata.ChainID != chainID {
			return fmt.Errorf("actor %s is configured for chain %s, expected %s", value.actor.Alias, metadata.ChainID, chainID)
		}
		if _, err := value.chain.ContractInfo(); err != nil {
			return fmt.Errorf("actor %s contract readiness: %w", value.actor.Alias, err)
		}
	}
	return nil
}

func (p *Pool) Status(ctx context.Context) (Status, error) {
	if err := ctx.Err(); err != nil {
		return Status{}, err
	}
	p.mu.RLock()
	aliases := make([]string, 0, len(p.clients))
	for alias := range p.clients {
		aliases = append(aliases, alias)
	}
	p.mu.RUnlock()
	sort.Strings(aliases)
	if len(aliases) == 0 {
		return Status{}, errors.New("actor client pool is empty")
	}
	first, _ := p.get(aliases[0])
	metadata, err := first.chain.Metadata()
	if err != nil {
		return Status{}, err
	}
	contractInfo, err := first.chain.ContractInfo()
	if err != nil {
		return Status{}, err
	}
	status := Status{Chain: metadata}
	status.Contract.Name = contractInfo.Name
	status.Contract.Version = contractInfo.Version
	status.Contract.Runtime = contractInfo.RuntimeType.String()
	status.Contract.Status = contractInfo.Status.String()
	for _, alias := range aliases {
		actor, _ := p.Actor(alias)
		status.Actors = append(status.Actors, actor)
	}
	return status, nil
}

func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, value := range p.clients {
		value.chain.Close()
	}
	p.clients = make(map[string]*actorClient)
}
