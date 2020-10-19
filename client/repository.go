// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package client

import "sync"

type Repository struct {
	Clients map[string]Client
	Mu      sync.RWMutex
}

// NewRepository creates a new Client Repo
func NewRepository() *Repository {
	return &Repository{
		Clients: make(map[string]Client),
	}
}
