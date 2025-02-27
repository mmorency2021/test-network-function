// Copyright (C) 2020-2021 Red Hat, Inc.
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, write to the Free Software Foundation, Inc.,
// 51 Franklin Street, Fifth Floor, Boston, MA 02110-1301 USA.

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	filePath = "testdata/tnf_test_config.yml"
)

func TestLoadConfigFromFile(t *testing.T) {
	env := GetTestEnvironment()
	assert.Nil(t, env.loadConfigFromFile(filePath))
	assert.NotNil(t, env.loadConfigFromFile(filePath)) // Loading when already loaded is an error case
	assert.Equal(t, env.Config.Partner.TestOrchestratorID.Namespace, "default")
	assert.Equal(t, env.Config.Partner.TestOrchestratorID.ContainerName, "partner")
	assert.Equal(t, env.Config.Partner.TestOrchestratorID.PodName, "partner")
}
