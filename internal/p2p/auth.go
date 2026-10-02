// Copyright 2026 Georg Hagn
// SPDX-License-Identifier: Apache-2.0

package p2p

import (
	"context"

	"codeberg.org/tiny-frameworks/nexutils/logger"
)

// NexfactUserStore encapsulates user management for nexfact-server.
type NexfactUserStore struct {
	// Here you can store later dependencies such as DB pointers (*sql.DB),
	// config files, or pharo services.
}

func NewNexfactUserStore() *NexfactUserStore {
	return &NexfactUserStore{}
}

// Authenticate verifies the user data for nexfact-server
func (s *NexfactUserStore) Authenticate(ctx context.Context, username, password string) bool {
	// Example check (later, e.g., hash comparison against database or config)
	if username == "admin" && password == "nexfact-secure-2026" {
		logger.Logger.Info("nexfact user authenticated successfully", "user", username)
		return true
	}

	logger.Logger.Warn("nexfact authentication failed", "user", username)
	return false
}
