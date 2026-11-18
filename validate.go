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
//   - transferServer requestBody.region: required
//   - postServerProperties: at least one property, autostop-time 3-1440, enum values
//   - backup name: at most 512 bytes; settings limit 2-1000, interval enum
//   - domains: port 1-65535, protocol https, at least one field on update
//   - access: username required, permissions from the known set
//
// IDs in tests/fixtures must therefore be 16 characters (e.g. the spec
// examples EwYiY9IAMtQBTb6U / X4iJREgqBGFSBtn0 / wdZhC3v1udSOxR1d).

const (
	specIDLength = 16
	minRamGB     = 2
	maxRamGB     = 16

	minAutoStopMinutes = 3
	maxAutoStopMinutes = 1440

	maxBackupNameBytes = 512
	minBackupLimit     = 2
	maxBackupLimit     = 1000

	minDomainPort = 1
	maxDomainPort = 65535
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

func validateRegion(region Region) error {
	if strings.TrimSpace(string(region)) == "" {
		return fmt.Errorf("invalid region: must not be empty")
	}
	return nil
}

func validateServerProperties(props ServerProperties) error {
	if props.AutoStop == nil && props.AutoStopTime == nil && props.AutoStart == nil &&
		props.AutoStartType == nil && props.LauncherAutoStart == nil &&
		props.LauncherAutoStartType == nil && props.EnableLobby == nil &&
		props.AutoUpdate == nil && props.ForceUpgrade == nil {
		return fmt.Errorf("no server properties provided")
	}
	if props.AutoStopTime != nil &&
		(*props.AutoStopTime < minAutoStopMinutes || *props.AutoStopTime > maxAutoStopMinutes) {
		return fmt.Errorf("invalid autostop-time %d: must be between %d and %d minutes",
			*props.AutoStopTime, minAutoStopMinutes, maxAutoStopMinutes)
	}
	if props.AutoStartType != nil {
		if err := validateAutoStartType(*props.AutoStartType); err != nil {
			return err
		}
	}
	if props.LauncherAutoStartType != nil {
		if err := validateAutoStartType(*props.LauncherAutoStartType); err != nil {
			return err
		}
	}
	if props.AutoUpdate != nil {
		if err := validateAutoUpdateMode(*props.AutoUpdate); err != nil {
			return err
		}
	}
	return nil
}

func validateAutoStartType(t AutoStartType) error {
	switch t {
	case AutoStartTypeAny, AutoStartTypeWhitelisted, AutoStartTypeOps:
		return nil
	default:
		return fmt.Errorf("invalid autostart type %q: must be %q, %q or %q",
			t, AutoStartTypeAny, AutoStartTypeWhitelisted, AutoStartTypeOps)
	}
}

func validateAutoUpdateMode(mode AutoUpdateMode) error {
	switch mode {
	case AutoUpdateNever, AutoUpdateAsk, AutoUpdateAlways:
		return nil
	default:
		return fmt.Errorf("invalid auto-update mode %q: must be %q, %q or %q",
			mode, AutoUpdateNever, AutoUpdateAsk, AutoUpdateAlways)
	}
}

func validateBackupID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("invalid backup ID: must not be empty")
	}
	return nil
}

func validateBackupName(name string) error {
	if len(name) > maxBackupNameBytes {
		return fmt.Errorf("invalid backup name: must be at most %d bytes", maxBackupNameBytes)
	}
	return nil
}

func validateBackupInterval(interval BackupInterval) error {
	switch interval {
	case BackupInterval6Hours, BackupInterval12Hours, BackupInterval24Hours, BackupInterval48Hours:
		return nil
	default:
		return fmt.Errorf("invalid backup interval %d: must be one of %d, %d, %d or %d",
			interval, BackupInterval6Hours, BackupInterval12Hours,
			BackupInterval24Hours, BackupInterval48Hours)
	}
}

func validateUpdateBackupSettings(opts UpdateBackupSettingsOptions) error {
	if opts.Enabled == nil && opts.Limit == nil && opts.Interval == nil {
		return fmt.Errorf("no backup settings provided")
	}
	if opts.Limit != nil && (*opts.Limit < minBackupLimit || *opts.Limit > maxBackupLimit) {
		return fmt.Errorf("invalid backup limit %d: must be between %d and %d",
			*opts.Limit, minBackupLimit, maxBackupLimit)
	}
	if opts.Interval != nil {
		if err := validateBackupInterval(*opts.Interval); err != nil {
			return err
		}
	}
	return nil
}

func validateDomainID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("invalid domain ID: must not be empty")
	}
	return nil
}

func validateDomainName(domain string) error {
	if strings.TrimSpace(domain) == "" {
		return fmt.Errorf("invalid domain: must not be empty")
	}
	if strings.Contains(domain, "://") || strings.ContainsAny(domain, " \t\r\n/") {
		return fmt.Errorf("invalid domain %q: must be a bare hostname", domain)
	}
	return nil
}

func validateDomainProtocol(protocol DomainProtocol) error {
	if protocol != DomainProtocolHTTPS {
		return fmt.Errorf("invalid domain protocol %q: must be %q", protocol, DomainProtocolHTTPS)
	}
	return nil
}

func validateDomainPort(port int) error {
	if port < minDomainPort || port > maxDomainPort {
		return fmt.Errorf("invalid port %d: must be between %d and %d",
			port, minDomainPort, maxDomainPort)
	}
	return nil
}

func validateDomainUpdate(opts UpdateDomainOptions) error {
	if opts.Protocol == "" && opts.Port == nil {
		return fmt.Errorf("at least one of protocol or port must be set")
	}
	if opts.Protocol != "" {
		if err := validateDomainProtocol(opts.Protocol); err != nil {
			return err
		}
	}
	if opts.Port != nil {
		return validateDomainPort(*opts.Port)
	}
	return nil
}

func validateAccessID(id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("invalid access ID: must not be empty")
	}
	return nil
}

func validateAccessUsername(username string) error {
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("invalid username: must not be empty")
	}
	return nil
}

func validatePermissions(permissions []Permission) error {
	for _, permission := range permissions {
		switch permission {
		case PermissionServer, PermissionStop, PermissionConsole, PermissionLog,
			PermissionOptions, PermissionReinstall, PermissionDomains,
			PermissionSoftware, PermissionPlayers, PermissionFiles, PermissionAddons,
			PermissionWorlds, PermissionBackups, PermissionTransactions,
			PermissionDuplicate, PermissionAccess:
		default:
			return fmt.Errorf("invalid permission %q", permission)
		}
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
