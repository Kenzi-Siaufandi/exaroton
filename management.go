package exaroton

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync/atomic"
)

// managementRequestID issues correlation IDs for management requests. An
// atomic counter is unique per process without the collision window of
// time-based IDs under concurrency.
var managementRequestID atomic.Uint64

func nextManagementID() string {
	return strconv.FormatUint(managementRequestID.Add(1), 10)
}

// Stats holds memory and CPU usage from the stats stream.
type Stats struct {
	Memory struct {
		Percent float64 `json:"percent"`
		Usage   int64   `json:"usage"`
	} `json:"memory"`
	CPU struct {
		Percent float64 `json:"percent"`
		Usage   float64 `json:"usage"`
		Limit   int     `json:"limit"`
	} `json:"cpu"`
}

type wsManagementRequest struct {
	ID     string `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params"`
}

// ManagementResponse is a correlated reply to SendManagementRequest.
type ManagementResponse struct {
	ID   string          `json:"id"`
	Data json.RawMessage `json:"data"`
}

// ManagementNotification is a server-pushed management event.
type ManagementNotification struct {
	Name string          `json:"name"`
	Data json.RawMessage `json:"data"`
}

// UntypedGameRule is a gamerule update payload without a type hint.
type UntypedGameRule struct {
	Value any    `json:"value"`
	Key   string `json:"key"`
}

// IncomingIPBan is an IP ban payload accepted by AddIPBans.
type IncomingIPBan struct {
	Reason  string `json:"reason"`
	Expires string `json:"expires"`
	IP      string `json:"ip"`
	Source  string `json:"source"`
	Player  Player `json:"player"`
}

// SystemMessage is a broadcast payload for SendSystemMessage.
type SystemMessage struct {
	ReceivingPlayers []Player `json:"receivingPlayers"`
	Overlay          bool     `json:"overlay"`
	Message          Message  `json:"message"`
}

// KickPlayer pairs a player with a kick message.
type KickPlayer struct {
	Player  Player  `json:"player"`
	Message Message `json:"message"`
}

// IPBan describes an IP ban entry.
type IPBan struct {
	Reason  string `json:"reason"`
	Expires string `json:"expires"`
	IP      string `json:"ip"`
	Source  string `json:"source"`
}

// TypedGameRule is a gamerule with its type hint.
type TypedGameRule struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
	Key   string `json:"key"`
}

// UserBan describes a player ban entry.
type UserBan struct {
	Reason  string `json:"reason"`
	Expires string `json:"expires"`
	Source  string `json:"source"`
	Player  Player `json:"player"`
}

// Message is a translatable or literal chat message.
type Message struct {
	Translatable       string   `json:"translatable"`
	TranslatableParams []string `json:"translatableParams"`
	Literal            string   `json:"literal"`
}

// Version describes the server version.
type Version struct {
	Protocol int    `json:"protocol"`
	Name     string `json:"name"`
}

// ServerState is the management status snapshot.
type ServerState struct {
	Players []Player `json:"players"`
	Started bool     `json:"started"`
	Version Version  `json:"version"`
}

// Operator describes an operator entry.
type Operator struct {
	PermissionLevel     int    `json:"permissionLevel,omitempty"`
	BypassesPlayerLimit *bool  `json:"bypassesPlayerLimit,omitempty"`
	Player              Player `json:"player"`
}

// Player identifies a Minecraft player by name and UUID.
type Player struct {
	Name string `json:"name"`
	UUID string `json:"id"`
}

// ManagementRequest is a minecraft:* management method name.
type ManagementRequest string

// ManagementNotificationName is a minecraft:notification/* event name.
type ManagementNotificationName string

const (
	// MethodAllowlistGet gets the allowlist
	MethodAllowlistGet ManagementRequest = "minecraft:allowlist"
	// MethodAllowlistSet sets the allowlist to the provided list of players
	MethodAllowlistSet ManagementRequest = "minecraft:allowlist/set"
	// MethodAllowlistAdd adds players to the allowlist
	MethodAllowlistAdd ManagementRequest = "minecraft:allowlist/add"
	// MethodAllowlistRemove removes players from the allowlist
	MethodAllowlistRemove ManagementRequest = "minecraft:allowlist/remove"
	// MethodAllowlistClear clears all players from the allowlist
	MethodAllowlistClear ManagementRequest = "minecraft:allowlist/clear"

	// MethodBansGet gets the ban list
	MethodBansGet ManagementRequest = "minecraft:bans"
	// MethodBansSet sets the ban list to the provided entries
	MethodBansSet ManagementRequest = "minecraft:bans/set"
	// MethodBansAdd adds a player to the ban list
	MethodBansAdd ManagementRequest = "minecraft:bans/add"
	// MethodBansRemove removes a player from the ban list
	MethodBansRemove ManagementRequest = "minecraft:bans/remove"
	// MethodBansClear clears all players in the ban list
	MethodBansClear ManagementRequest = "minecraft:bans/clear"

	// MethodIPBansGet gets the IP ban list
	MethodIPBansGet ManagementRequest = "minecraft:ip_bans"
	// MethodIPBansSet sets the IP ban list
	MethodIPBansSet ManagementRequest = "minecraft:ip_bans/set"
	// MethodIPBansAdd adds an IP to the ban list
	MethodIPBansAdd ManagementRequest = "minecraft:ip_bans/add"
	// MethodIPBansRemove removes an IP from the ban list
	MethodIPBansRemove ManagementRequest = "minecraft:ip_bans/remove"
	// MethodIPBansClear clears all IP bans in the ban list
	MethodIPBansClear ManagementRequest = "minecraft:ip_bans/clear"

	// MethodPlayersGet gets the list of currently connected players
	MethodPlayersGet ManagementRequest = "minecraft:players"
	// MethodPlayersKick kicks a player from the server
	MethodPlayersKick ManagementRequest = "minecraft:players/kick"

	// MethodOperatorsGet gets the current operator list
	MethodOperatorsGet ManagementRequest = "minecraft:operators"
	// MethodOperatorsSet sets the operator list
	MethodOperatorsSet ManagementRequest = "minecraft:operators/set"
	// MethodOperatorsAdd adds a player as an operator
	MethodOperatorsAdd ManagementRequest = "minecraft:operators/add"
	// MethodOperatorsRemove removes operator status from a player
	MethodOperatorsRemove ManagementRequest = "minecraft:operators/remove"
	// MethodOperatorsClear clears all operators
	MethodOperatorsClear ManagementRequest = "minecraft:operators/clear"

	// MethodServerStatus gets general server health and status
	MethodServerStatus ManagementRequest = "minecraft:server/status"
	// MethodServerSave triggers a world save
	MethodServerSave ManagementRequest = "minecraft:server/save"
	// MethodServerStop stops the server
	MethodServerStop ManagementRequest = "minecraft:server/stop"
	// MethodServerSystemMessage sends a system-level broadcast
	MethodServerSystemMessage ManagementRequest = "minecraft:server/system_message"

	// MethodServerSettingsAutoSave gets whether automatic world saving is enabled
	MethodServerSettingsAutoSave ManagementRequest = "minecraft:serversettings/autosave"
	// MethodServerSettingsAutoSaveSet enables or disables automatic world saving
	MethodServerSettingsAutoSaveSet ManagementRequest = "minecraft:serversettings/autosave/set"
	// MethodServerSettingsDifficulty gets the current difficulty level
	MethodServerSettingsDifficulty ManagementRequest = "minecraft:serversettings/difficulty"
	// MethodServerSettingsDifficultySet sets the difficulty level
	MethodServerSettingsDifficultySet ManagementRequest = "minecraft:serversettings/difficulty/set"
	// MethodServerSettingsEnforceAllowlist gets whether allowlist enforcement is enabled
	MethodServerSettingsEnforceAllowlist ManagementRequest = "minecraft:serversettings/enforce_allowlist"
	// MethodServerSettingsEnforceAllowlistSet enables or disables allowlist enforcement
	MethodServerSettingsEnforceAllowlistSet ManagementRequest = "minecraft:serversettings/enforce_allowlist/set"
	// MethodServerSettingsUseAllowlist gets whether the allowlist is enabled
	MethodServerSettingsUseAllowlist ManagementRequest = "minecraft:serversettings/use_allowlist"
	// MethodServerSettingsUseAllowlistSet enables or disables the allowlist
	MethodServerSettingsUseAllowlistSet ManagementRequest = "minecraft:serversettings/use_allowlist/set"
	// MethodServerSettingsMaxPlayers gets the maximum number of players
	MethodServerSettingsMaxPlayers ManagementRequest = "minecraft:serversettings/max_players"
	// MethodServerSettingsMaxPlayersSet sets the maximum number of players
	MethodServerSettingsMaxPlayersSet ManagementRequest = "minecraft:serversettings/max_players/set"
	// MethodServerSettingsPauseWhenEmptySeconds gets seconds before auto-pause when empty
	MethodServerSettingsPauseWhenEmptySeconds ManagementRequest = "minecraft:serversettings/pause_when_empty_seconds"
	// MethodServerSettingsPauseWhenEmptySecondsSet sets seconds before auto-pause when empty
	MethodServerSettingsPauseWhenEmptySecondsSet ManagementRequest = "minecraft:serversettings/pause_when_empty_seconds/set"
	// MethodServerSettingsPlayerIdleTimeout gets seconds before idle players are kicked
	MethodServerSettingsPlayerIdleTimeout ManagementRequest = "minecraft:serversettings/player_idle_timeout"
	// MethodServerSettingsPlayerIdleTimeoutSet sets seconds before idle players are kicked
	MethodServerSettingsPlayerIdleTimeoutSet ManagementRequest = "minecraft:serversettings/player_idle_timeout/set"
	// MethodServerSettingsAllowFlight gets whether flight is allowed in Survival mode
	MethodServerSettingsAllowFlight ManagementRequest = "minecraft:serversettings/allow_flight"
	// MethodServerSettingsAllowFlightSet sets whether flight is allowed in Survival mode
	MethodServerSettingsAllowFlightSet ManagementRequest = "minecraft:serversettings/allow_flight/set"
	// MethodServerSettingsMotd gets the server message of the day
	MethodServerSettingsMotd ManagementRequest = "minecraft:serversettings/motd"
	// MethodServerSettingsMotdSet sets the server message of the day
	MethodServerSettingsMotdSet ManagementRequest = "minecraft:serversettings/motd/set"
	// MethodServerSettingsSpawnProtectionRadius gets the spawn protection radius in blocks
	MethodServerSettingsSpawnProtectionRadius ManagementRequest = "minecraft:serversettings/spawn_protection_radius"
	// MethodServerSettingsSpawnProtectionRadiusSet sets the spawn protection radius in blocks
	MethodServerSettingsSpawnProtectionRadiusSet ManagementRequest = "minecraft:serversettings/spawn_protection_radius/set"
	// MethodServerSettingsForceGameMode gets whether players are forced into the default game mode
	MethodServerSettingsForceGameMode ManagementRequest = "minecraft:serversettings/force_game_mode"
	// MethodServerSettingsForceGameModeSet sets whether players are forced into the default game mode
	MethodServerSettingsForceGameModeSet ManagementRequest = "minecraft:serversettings/force_game_mode/set"
	// MethodServerSettingsGameMode gets the default game mode
	MethodServerSettingsGameMode ManagementRequest = "minecraft:serversettings/game_mode"
	// MethodServerSettingsGameModeSet sets the default game mode
	MethodServerSettingsGameModeSet ManagementRequest = "minecraft:serversettings/game_mode/set"
	// MethodServerSettingsViewDistance gets the view distance in chunks
	MethodServerSettingsViewDistance ManagementRequest = "minecraft:serversettings/view_distance"
	// MethodServerSettingsViewDistanceSet sets the view distance in chunks
	MethodServerSettingsViewDistanceSet ManagementRequest = "minecraft:serversettings/view_distance/set"
	// MethodServerSettingsSimulationDistance gets the simulation distance in chunks
	MethodServerSettingsSimulationDistance ManagementRequest = "minecraft:serversettings/simulation_distance"
	// MethodServerSettingsSimulationDistanceSet sets the simulation distance in chunks
	MethodServerSettingsSimulationDistanceSet ManagementRequest = "minecraft:serversettings/simulation_distance/set"
	// MethodServerSettingsAcceptTransfers gets whether the server accepts player transfers
	MethodServerSettingsAcceptTransfers ManagementRequest = "minecraft:serversettings/accept_transfers"
	// MethodServerSettingsAcceptTransfersSet sets whether the server accepts player transfers
	MethodServerSettingsAcceptTransfersSet ManagementRequest = "minecraft:serversettings/accept_transfers/set"
	// MethodServerSettingsStatusHeartbeatInterval gets the interval between status heartbeats
	MethodServerSettingsStatusHeartbeatInterval ManagementRequest = "minecraft:serversettings/status_heartbeat_interval"
	// MethodServerSettingsStatusHeartbeatIntervalSet sets the interval between status heartbeats
	MethodServerSettingsStatusHeartbeatIntervalSet ManagementRequest = "minecraft:serversettings/status_heartbeat_interval/set"
	// MethodServerSettingsOperatorUserPermissionLevel gets the permission level for operator commands
	MethodServerSettingsOperatorUserPermissionLevel ManagementRequest = "minecraft:serversettings/operator_user_permission_level"
	// MethodServerSettingsOperatorUserPermissionLevelSet sets the permission level for operator commands
	MethodServerSettingsOperatorUserPermissionLevelSet ManagementRequest = "minecraft:serversettings/operator_user_permission_level/set"
	// MethodServerSettingsHideOnlinePlayers gets whether online players are hidden from status queries
	MethodServerSettingsHideOnlinePlayers ManagementRequest = "minecraft:serversettings/hide_online_players"
	// MethodServerSettingsHideOnlinePlayersSet sets whether online players are hidden from status queries
	MethodServerSettingsHideOnlinePlayersSet ManagementRequest = "minecraft:serversettings/hide_online_players/set"
	// MethodServerSettingsStatusReplies gets whether the server responds to status requests
	MethodServerSettingsStatusReplies ManagementRequest = "minecraft:serversettings/status_replies"
	// MethodServerSettingsStatusRepliesSet sets whether the server responds to status requests
	MethodServerSettingsStatusRepliesSet ManagementRequest = "minecraft:serversettings/status_replies/set"
	// MethodServerSettingsEntityBroadcastRange gets the entity broadcast range percentage
	MethodServerSettingsEntityBroadcastRange ManagementRequest = "minecraft:serversettings/entity_broadcast_range"
	// MethodServerSettingsEntityBroadcastRangeSet sets the entity broadcast range percentage
	MethodServerSettingsEntityBroadcastRangeSet ManagementRequest = "minecraft:serversettings/entity_broadcast_range/set"

	// MethodGameRulesGet gets the gamerule keys and current values
	MethodGameRulesGet ManagementRequest = "minecraft:gamerules"
	// MethodGameRulesUpdate updates one gamerule value
	MethodGameRulesUpdate ManagementRequest = "minecraft:gamerules/update"
)

const (
	// NotificationServerStarted fires when the server started. No payload.
	NotificationServerStarted ManagementNotificationName = "minecraft:notification/server/started"
	// NotificationServerStopping fires when the server is shutting down. No payload.
	NotificationServerStopping ManagementNotificationName = "minecraft:notification/server/stopping"
	// NotificationServerSaving fires when a server save started. No payload.
	NotificationServerSaving ManagementNotificationName = "minecraft:notification/server/saving"
	// NotificationServerSaved fires when a server save completed. No payload.
	NotificationServerSaved ManagementNotificationName = "minecraft:notification/server/saved"
	// NotificationServerStatus is the server status heartbeat. Payload is ServerState.
	NotificationServerStatus ManagementNotificationName = "minecraft:notification/server/status"
	// NotificationServerActivity fires when the network connection initialized. No payload.
	NotificationServerActivity ManagementNotificationName = "minecraft:notification/server/activity"

	// NotificationPlayersJoined fires when a player joined. Payload is Player.
	NotificationPlayersJoined ManagementNotificationName = "minecraft:notification/players/joined"
	// NotificationPlayersLeft fires when a player left. Payload is Player.
	NotificationPlayersLeft ManagementNotificationName = "minecraft:notification/players/left"

	// NotificationOperatorsAdded fires when a player was oped. Payload is Operator.
	NotificationOperatorsAdded ManagementNotificationName = "minecraft:notification/operators/added"
	// NotificationOperatorsRemoved fires when a player was deoped. Payload is Operator.
	NotificationOperatorsRemoved ManagementNotificationName = "minecraft:notification/operators/removed"

	// NotificationAllowlistAdded fires when a player was added to the allowlist. Payload is Player.
	NotificationAllowlistAdded ManagementNotificationName = "minecraft:notification/allowlist/added"
	// NotificationAllowlistRemoved fires when a player was removed from the allowlist. Payload is Player.
	NotificationAllowlistRemoved ManagementNotificationName = "minecraft:notification/allowlist/removed"

	// NotificationIPBansAdded fires when an IP was added to the ban list. Payload is IPBan.
	NotificationIPBansAdded ManagementNotificationName = "minecraft:notification/ip_bans/added"
	// NotificationIPBansRemoved fires when an IP was removed from the ban list. Payload is a string IP.
	NotificationIPBansRemoved ManagementNotificationName = "minecraft:notification/ip_bans/removed"

	// NotificationBansAdded fires when a player was added to the ban list. Payload is UserBan.
	NotificationBansAdded ManagementNotificationName = "minecraft:notification/bans/added"
	// NotificationBansRemoved fires when a player was removed from the ban list. Payload is Player.
	NotificationBansRemoved ManagementNotificationName = "minecraft:notification/bans/removed"

	// NotificationGameRuleUpdated fires when a gamerule changed. Payload is TypedGameRule.
	NotificationGameRuleUpdated ManagementNotificationName = "minecraft:notification/gamerules/updated"

	// NotificationWorldUpgradeStarted fires when a world upgrade started. No payload.
	NotificationWorldUpgradeStarted ManagementNotificationName = "minecraft:notification/world/upgrade_started"
	// NotificationWorldUpgradeProgress reports world upgrade progress (0-1, rate limited to 1/sec).
	NotificationWorldUpgradeProgress ManagementNotificationName = "minecraft:notification/world/upgrade_progress"
	// NotificationWorldUpgradeFinished fires when a world upgrade finished. No payload.
	NotificationWorldUpgradeFinished ManagementNotificationName = "minecraft:notification/world/upgrade_finished"
	// NotificationWorldUpgradeFailed fires when a world upgrade failed. Payload is a reason string.
	NotificationWorldUpgradeFailed ManagementNotificationName = "minecraft:notification/world/upgrade_failed"
)

// sendManagement sends a management request, waits for the correlated
// response, and decodes its data payload into T.
func sendManagement[T any](s *Session, ctx context.Context, method ManagementRequest, params any) (T, error) {
	var zero T
	rawJSON, err := s.sendRequestAndWait(ctx, method, params)
	if err != nil {
		return zero, err
	}
	var out T
	if err := json.Unmarshal(rawJSON, &out); err != nil {
		return zero, err
	}
	return out, nil
}

// GetAllowlist returns the allowlist.
func (s *Session) GetAllowlist(ctx context.Context) ([]Player, error) {
	return sendManagement[[]Player](s, ctx, MethodAllowlistGet, []any{})
}

// SetAllowlist replaces the allowlist and returns the updated list.
func (s *Session) SetAllowlist(ctx context.Context, players []Player) ([]Player, error) {
	return sendManagement[[]Player](s, ctx, MethodAllowlistSet, []any{players})
}

// AddAllowlist adds players to the allowlist and returns the updated list.
func (s *Session) AddAllowlist(ctx context.Context, players []Player) ([]Player, error) {
	return sendManagement[[]Player](s, ctx, MethodAllowlistAdd, []any{players})
}

// RemoveAllowlist removes players from the allowlist and returns the updated list.
func (s *Session) RemoveAllowlist(ctx context.Context, players []Player) ([]Player, error) {
	return sendManagement[[]Player](s, ctx, MethodAllowlistRemove, []any{players})
}

// ClearAllowlist clears all players in the allowlist and returns the updated list.
func (s *Session) ClearAllowlist(ctx context.Context) ([]Player, error) {
	return sendManagement[[]Player](s, ctx, MethodAllowlistClear, []any{})
}

// GetBans returns the ban list.
func (s *Session) GetBans(ctx context.Context) ([]UserBan, error) {
	return sendManagement[[]UserBan](s, ctx, MethodBansGet, []any{})
}

// SetBans replaces the ban list and returns the updated list.
func (s *Session) SetBans(ctx context.Context, bans []UserBan) ([]UserBan, error) {
	return sendManagement[[]UserBan](s, ctx, MethodBansSet, []any{bans})
}

// AddBans adds players to the ban list and returns the updated list.
func (s *Session) AddBans(ctx context.Context, bans []UserBan) ([]UserBan, error) {
	return sendManagement[[]UserBan](s, ctx, MethodBansAdd, []any{bans})
}

// RemoveBans removes players from the ban list and returns the updated list.
func (s *Session) RemoveBans(ctx context.Context, players []Player) ([]UserBan, error) {
	return sendManagement[[]UserBan](s, ctx, MethodBansRemove, []any{players})
}

// ClearBans clears all players in the ban list and returns the updated list.
func (s *Session) ClearBans(ctx context.Context) ([]UserBan, error) {
	return sendManagement[[]UserBan](s, ctx, MethodBansClear, []any{})
}

// GetIPBans returns the IP ban list.
func (s *Session) GetIPBans(ctx context.Context) ([]IPBan, error) {
	return sendManagement[[]IPBan](s, ctx, MethodIPBansGet, []any{})
}

// SetIPBans replaces the IP ban list and returns the updated list.
func (s *Session) SetIPBans(ctx context.Context, bans []IPBan) ([]IPBan, error) {
	return sendManagement[[]IPBan](s, ctx, MethodIPBansSet, []any{bans})
}

// AddIPBans adds IPs to the ban list and returns the updated list.
func (s *Session) AddIPBans(ctx context.Context, bans []IncomingIPBan) ([]IPBan, error) {
	return sendManagement[[]IPBan](s, ctx, MethodIPBansAdd, []any{bans})
}

// RemoveIPBans removes IPs from the ban list and returns the updated list.
func (s *Session) RemoveIPBans(ctx context.Context, ips []string) ([]IPBan, error) {
	return sendManagement[[]IPBan](s, ctx, MethodIPBansRemove, []any{ips})
}

// ClearIPBans clears all IPs in the ban list and returns the updated list.
func (s *Session) ClearIPBans(ctx context.Context) ([]IPBan, error) {
	return sendManagement[[]IPBan](s, ctx, MethodIPBansClear, []any{})
}

// GetPlayers returns all connected players.
func (s *Session) GetPlayers(ctx context.Context) ([]Player, error) {
	return sendManagement[[]Player](s, ctx, MethodPlayersGet, []any{})
}

// KickPlayers kicks players and returns the kicked players.
func (s *Session) KickPlayers(ctx context.Context, kicks []KickPlayer) ([]Player, error) {
	return sendManagement[[]Player](s, ctx, MethodPlayersKick, []any{kicks})
}

// GetOperators returns all oped players.
func (s *Session) GetOperators(ctx context.Context) ([]Operator, error) {
	return sendManagement[[]Operator](s, ctx, MethodOperatorsGet, []any{})
}

// SetOperators replaces the operator list and returns the updated list.
func (s *Session) SetOperators(ctx context.Context, ops []Operator) ([]Operator, error) {
	return sendManagement[[]Operator](s, ctx, MethodOperatorsSet, []any{ops})
}

// AddOperators adds operators and returns the updated list.
func (s *Session) AddOperators(ctx context.Context, ops []Operator) ([]Operator, error) {
	return sendManagement[[]Operator](s, ctx, MethodOperatorsAdd, []any{ops})
}

// RemoveOperators removes operators and returns the updated list.
func (s *Session) RemoveOperators(ctx context.Context, players []Player) ([]Operator, error) {
	return sendManagement[[]Operator](s, ctx, MethodOperatorsRemove, []any{players})
}

// ClearOperators clears all operators and returns the updated list.
func (s *Session) ClearOperators(ctx context.Context) ([]Operator, error) {
	return sendManagement[[]Operator](s, ctx, MethodOperatorsClear, []any{})
}

// GetServerStatus returns the live server status snapshot.
func (s *Session) GetServerStatus(ctx context.Context) (*ServerState, error) {
	return sendManagement[*ServerState](s, ctx, MethodServerStatus, []any{})
}

// SaveServer saves the server state. flush selects whether data is flushed to disk.
func (s *Session) SaveServer(ctx context.Context, flush bool) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSave, []any{flush})
}

// StopServer stops the server and reports whether stopping began.
func (s *Session) StopServer(ctx context.Context) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerStop, []any{})
}

// SendSystemMessage sends a system message and reports whether it was sent.
func (s *Session) SendSystemMessage(ctx context.Context, message SystemMessage) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSystemMessage, []any{message})
}

// GetAutoSave returns whether automatic world saving is enabled.
func (s *Session) GetAutoSave(ctx context.Context) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsAutoSave, []any{})
}

// SetAutoSave enables or disables automatic world saving and returns the new value.
func (s *Session) SetAutoSave(ctx context.Context, enable bool) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsAutoSaveSet, []any{enable})
}

// GetDifficulty returns the current difficulty level.
func (s *Session) GetDifficulty(ctx context.Context) (string, error) {
	return sendManagement[string](s, ctx, MethodServerSettingsDifficulty, []any{})
}

// SetDifficulty sets the difficulty level and returns the new value.
func (s *Session) SetDifficulty(ctx context.Context, difficulty string) (string, error) {
	return sendManagement[string](s, ctx, MethodServerSettingsDifficultySet, []any{difficulty})
}

// GetEnforceAllowlist returns whether allowlist enforcement is enabled (kicks on removal).
func (s *Session) GetEnforceAllowlist(ctx context.Context) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsEnforceAllowlist, []any{})
}

// SetEnforceAllowlist enables or disables allowlist enforcement and returns the new value.
func (s *Session) SetEnforceAllowlist(ctx context.Context, enforce bool) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsEnforceAllowlistSet, []any{enforce})
}

// GetUseAllowlist returns whether the allowlist is enabled.
func (s *Session) GetUseAllowlist(ctx context.Context) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsUseAllowlist, []any{})
}

// SetUseAllowlist enables or disables the allowlist and returns the new value.
func (s *Session) SetUseAllowlist(ctx context.Context, use bool) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsUseAllowlistSet, []any{use})
}

// GetMaxPlayers returns the maximum number of players allowed.
func (s *Session) GetMaxPlayers(ctx context.Context) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsMaxPlayers, []any{})
}

// SetMaxPlayers sets the maximum number of players and returns the new value.
func (s *Session) SetMaxPlayers(ctx context.Context, max int) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsMaxPlayersSet, []any{max})
}

// GetPauseWhenEmptySeconds returns seconds before auto-pause when no players are online.
func (s *Session) GetPauseWhenEmptySeconds(ctx context.Context) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsPauseWhenEmptySeconds, []any{})
}

// SetPauseWhenEmptySeconds sets seconds before auto-pause when empty and returns the new value.
func (s *Session) SetPauseWhenEmptySeconds(ctx context.Context, seconds int) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsPauseWhenEmptySecondsSet, []any{seconds})
}

// GetPlayerIdleTimeout returns seconds before idle players are kicked.
func (s *Session) GetPlayerIdleTimeout(ctx context.Context) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsPlayerIdleTimeout, []any{})
}

// SetPlayerIdleTimeout sets seconds before idle players are kicked and returns the new value.
func (s *Session) SetPlayerIdleTimeout(ctx context.Context, seconds int) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsPlayerIdleTimeoutSet, []any{seconds})
}

// GetAllowFlight returns whether flight is allowed in Survival mode.
func (s *Session) GetAllowFlight(ctx context.Context) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsAllowFlight, []any{})
}

// SetAllowFlight sets whether flight is allowed in Survival mode and returns the new value.
func (s *Session) SetAllowFlight(ctx context.Context, allowed bool) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsAllowFlightSet, []any{allowed})
}

// GetMotd returns the server message of the day.
func (s *Session) GetMotd(ctx context.Context) (string, error) {
	return sendManagement[string](s, ctx, MethodServerSettingsMotd, []any{})
}

// SetMotd sets the server message of the day and returns the new value.
func (s *Session) SetMotd(ctx context.Context, message string) (string, error) {
	return sendManagement[string](s, ctx, MethodServerSettingsMotdSet, []any{message})
}

// GetSpawnProtectionRadius returns the spawn protection radius in blocks.
func (s *Session) GetSpawnProtectionRadius(ctx context.Context) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsSpawnProtectionRadius, []any{})
}

// SetSpawnProtectionRadius sets the spawn protection radius in blocks and returns the new value.
func (s *Session) SetSpawnProtectionRadius(ctx context.Context, radius int) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsSpawnProtectionRadiusSet, []any{radius})
}

// GetForceGameMode returns whether players are forced into the default game mode.
func (s *Session) GetForceGameMode(ctx context.Context) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsForceGameMode, []any{})
}

// SetForceGameMode sets whether players are forced into the default game mode and returns the new value.
func (s *Session) SetForceGameMode(ctx context.Context, force bool) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsForceGameModeSet, []any{force})
}

// GetGameMode returns the default game mode.
func (s *Session) GetGameMode(ctx context.Context) (string, error) {
	return sendManagement[string](s, ctx, MethodServerSettingsGameMode, []any{})
}

// SetGameMode sets the default game mode and returns the new value.
func (s *Session) SetGameMode(ctx context.Context, mode string) (string, error) {
	return sendManagement[string](s, ctx, MethodServerSettingsGameModeSet, []any{mode})
}

// GetViewDistance returns the view distance in chunks.
func (s *Session) GetViewDistance(ctx context.Context) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsViewDistance, []any{})
}

// SetViewDistance sets the view distance in chunks and returns the new value.
func (s *Session) SetViewDistance(ctx context.Context, distance int) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsViewDistanceSet, []any{distance})
}

// GetSimulationDistance returns the simulation distance in chunks.
func (s *Session) GetSimulationDistance(ctx context.Context) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsSimulationDistance, []any{})
}

// SetSimulationDistance sets the simulation distance in chunks and returns the new value.
func (s *Session) SetSimulationDistance(ctx context.Context, distance int) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsSimulationDistanceSet, []any{distance})
}

// GetAcceptTransfers returns whether the server accepts player transfers.
func (s *Session) GetAcceptTransfers(ctx context.Context) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsAcceptTransfers, []any{})
}

// SetAcceptTransfers sets whether the server accepts player transfers and returns the new value.
func (s *Session) SetAcceptTransfers(ctx context.Context, accept bool) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsAcceptTransfersSet, []any{accept})
}

// GetStatusHeartbeatInterval returns the interval in seconds between status heartbeats.
func (s *Session) GetStatusHeartbeatInterval(ctx context.Context) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsStatusHeartbeatInterval, []any{})
}

// SetStatusHeartbeatInterval sets the interval in seconds between status heartbeats and returns the new value.
func (s *Session) SetStatusHeartbeatInterval(ctx context.Context, seconds int) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsStatusHeartbeatIntervalSet, []any{seconds})
}

// GetOperatorUserPermissionLevel returns the permission level required for operator commands.
func (s *Session) GetOperatorUserPermissionLevel(ctx context.Context) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsOperatorUserPermissionLevel, []any{})
}

// SetOperatorUserPermissionLevel sets the permission level for operator commands and returns the new value.
func (s *Session) SetOperatorUserPermissionLevel(ctx context.Context, level int) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsOperatorUserPermissionLevelSet, []any{level})
}

// GetHideOnlinePlayers returns whether online players are hidden from status queries.
func (s *Session) GetHideOnlinePlayers(ctx context.Context) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsHideOnlinePlayers, []any{})
}

// SetHideOnlinePlayers sets whether online players are hidden from status queries and returns the new value.
func (s *Session) SetHideOnlinePlayers(ctx context.Context, hide bool) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsHideOnlinePlayersSet, []any{hide})
}

// GetStatusReplies returns whether the server responds to status requests.
func (s *Session) GetStatusReplies(ctx context.Context) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsStatusReplies, []any{})
}

// SetStatusReplies sets whether the server responds to status requests and returns the new value.
func (s *Session) SetStatusReplies(ctx context.Context, enable bool) (bool, error) {
	return sendManagement[bool](s, ctx, MethodServerSettingsStatusRepliesSet, []any{enable})
}

// GetEntityBroadcastRange returns the entity broadcast range percentage.
func (s *Session) GetEntityBroadcastRange(ctx context.Context) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsEntityBroadcastRange, []any{})
}

// SetEntityBroadcastRange sets the entity broadcast range percentage and returns the new value.
func (s *Session) SetEntityBroadcastRange(ctx context.Context, percentagePoints int) (int, error) {
	return sendManagement[int](s, ctx, MethodServerSettingsEntityBroadcastRangeSet, []any{percentagePoints})
}

// GetGameRules returns the gamerule keys and current values.
func (s *Session) GetGameRules(ctx context.Context) ([]TypedGameRule, error) {
	return sendManagement[[]TypedGameRule](s, ctx, MethodGameRulesGet, []any{})
}

// UpdateGameRule updates one gamerule value and returns the typed gamerule.
func (s *Session) UpdateGameRule(ctx context.Context, rule UntypedGameRule) (*TypedGameRule, error) {
	return sendManagement[*TypedGameRule](s, ctx, MethodGameRulesUpdate, []any{rule})
}

// SendManagementRequest sends a raw fire-and-forget management request and
// returns its correlation ID. The request is not registered as pending, so
// its response arrives on ManagementResponseChan for manual correlation.
// Prefer the typed helpers.
func (s *Session) SendManagementRequest(ctx context.Context, method ManagementRequest, params any) (string, error) {
	id := nextManagementID()
	if params == nil {
		params = []any{}
	}

	req := wsManagementRequest{
		ID:     id,
		Method: string(method),
		Params: params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}

	err = s.Send(ctx, wsEnvelope{
		Stream: "management",
		Type:   "request",
		Data:   data,
	})

	return id, err
}

func (s *Session) sendRequestAndWait(ctx context.Context, method ManagementRequest, params any) (json.RawMessage, error) {
	id := nextManagementID()
	if params == nil {
		params = []any{}
	}

	respChan := make(chan *ManagementResponse, 1)

	s.pendingMu.Lock()
	s.pendingRequests[id] = respChan
	s.pendingMu.Unlock()

	// Only delete from the map. The channel is never closed: routeMessage
	// looks it up under pendingMu then sends after unlocking, so closing here
	// races with a late response and panics (send on closed channel). The
	// buffer (size 1) plus non-blocking send means an orphaned channel after
	// ctx expiry is simply dropped and garbage-collected.
	defer func() {
		s.pendingMu.Lock()
		delete(s.pendingRequests, id)
		s.pendingMu.Unlock()
	}()

	req := wsManagementRequest{
		ID:     id,
		Method: string(method),
		Params: params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	err = s.Send(ctx, wsEnvelope{
		Stream: "management",
		Type:   "request",
		Data:   data,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	select {
	case response := <-respChan:
		var errResp struct {
			Error *wsError `json:"error"`
		}

		if err := json.Unmarshal(response.Data, &errResp); err == nil && errResp.Error != nil {
			return nil, errResp.Error
		} else {
			return response.Data, nil
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-s.done:
		return nil, fmt.Errorf("request failed: connection closed (the server might be offline)")
	}
}
