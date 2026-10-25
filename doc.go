// Package exaroton is a Go client for the exaroton API.
//
// REST and websocket/management types are hand-written: Server, Account,
// CreditPool, Software, file/config types plus Session streams (console,
// tick, stats, heap, status) and management requests/notifications.
// Server lifecycle includes creation, deletion and software installation.
//
// Validation mirrors the API (server/pool/software IDs are 16 chars, file
// paths are relative without . or .. segments, RAM is 2-16 GB) and fails
// fast before network I/O. REST errors are *ApiError with StatusCode for
// errors.As inspection.
//
// Websocket delivery is drop-on-full (buffered channels, non-blocking sends)
// with no automatic reconnect. Ping/pong with read deadlines keeps idle
// connections healthy; when the connection closes, channels are closed and
// the caller must dial again with ServerRef.OpenWebSocket.
package exaroton
