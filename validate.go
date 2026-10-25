package exaroton

import (
	"fmt"
	"strings"
)

// Validation constraints mirror the API so invalid requests
// fail fast before network I/O:
//
//   - components.parameters.serverId / poolId / softwareId: minLength 16, maxLength 16
//   - components.parameters.filePath: minLength 1
//   - postServerRam requestBody.ram: minimum 2, maximum 16
//   - postServer requestBody.name: required, plus software or softwareType
//
// IDs in tests/fixtures must therefore be 16 characters (e.g. the spec
// examples EwYiY9IAMtQBTb6U / X4iJREgqBGFSBtn0 / wdZhC3v1udSOxR1d).

const (
	specIDLength = 16
	minRamGB     = 2
	maxRamGB     = 16
)

func validateServerID(id string) error {
	if len(id) != specIDLength {
		return fmt.Errorf("invalid server ID %q: must be %d characters", id, specIDLength)
	}
	return nil
}

func validatePoolID(id string) error {
	if len(id) != specIDLength {
		return fmt.Errorf("invalid credit pool ID %q: must be %d characters", id, specIDLength)
	}
	return nil
}

func validateSoftwareID(id string) error {
	if len(id) != specIDLength {
		return fmt.Errorf("invalid software ID %q: must be %d characters", id, specIDLength)
	}
	return nil
}

func validateCreateServer(opts CreateServerOptions) error {
	if strings.TrimSpace(opts.Name) == "" {
		return fmt.Errorf("invalid server name: must not be empty")
	}
	if opts.Software == "" && opts.SoftwareType == "" {
		return fmt.Errorf("either software or softwareType must be set")
	}
	if opts.Software != "" {
		return validateSoftwareID(opts.Software)
	}
	return nil
}

func validateFilePath(p string) error {
	if len(p) < 1 {
		return fmt.Errorf("invalid file path: must not be empty")
	}
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("invalid file path %q: must be relative", p)
	}
	for _, seg := range strings.Split(strings.TrimSuffix(p, "/"), "/") {
		if seg == "" || seg == "." || seg == ".." {
			return fmt.Errorf("invalid file path %q: must not contain empty, . or .. segments", p)
		}
	}
	return nil
}

func validateRam(gb int) error {
	if gb < minRamGB || gb > maxRamGB {
		return fmt.Errorf("invalid ram %d: must be between %d and %d GB", gb, minRamGB, maxRamGB)
	}
	return nil
}

func (s *ServerRef) validate() error {
	if s == nil || s.client == nil {
		return fmt.Errorf("invalid server reference")
	}
	return validateServerID(s.serverID)
}

func (c *CreditRef) validate() error {
	if c == nil || c.client == nil {
		return fmt.Errorf("invalid credit pool reference")
	}
	return validatePoolID(c.poolID)
}

func (f *FileRef) validate() error {
	if f == nil || f.client == nil {
		return fmt.Errorf("invalid file reference")
	}
	if err := validateServerID(f.serverID); err != nil {
		return err
	}
	return validateFilePath(f.path)
}

func (l *PlayerListRef) validate() error {
	if l == nil || l.client == nil {
		return fmt.Errorf("invalid player list reference")
	}
	if err := validateServerID(l.serverID); err != nil {
		return err
	}
	if l.listName == "" {
		return fmt.Errorf("invalid player list name: must not be empty")
	}
	return nil
}
