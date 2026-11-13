/*
Copyright 2026 The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dbclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/models"
)

func TestServiceBusService(t *testing.T) {
	const (
		urlA = "http://127.0.0.1:9000/a"
		urlB = "http://127.0.0.1:9000/b"
	)

	t.Run("uses the edgecore database", func(t *testing.T) {
		s := NewServiceBusService()
		require.NoError(t, s.InsertUrls(urlA))
		t.Cleanup(func() {
			require.NoError(t, s.DeleteUrlsByKey(urlA))
		})

		target, err := s.GetUrlsByKey(urlA)
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.Equal(t, urlA, target.URL)
	})

	t.Run("insert, get and delete urls", func(t *testing.T) {
		s := &ServiceBusService{db: newTestDB(t, &models.TargetUrls{})}
		assert.True(t, s.IsTableEmpty())

		require.NoError(t, s.InsertUrls(urlA))
		require.NoError(t, s.InsertUrls(urlB))
		// Inserting an existing url replaces it instead of adding a duplicate.
		require.NoError(t, s.InsertUrls(urlA))
		assert.False(t, s.IsTableEmpty())
		var count int64
		require.NoError(t, s.db.Model(&models.TargetUrls{}).Count(&count).Error)
		assert.Equal(t, int64(2), count)

		target, err := s.GetUrlsByKey(urlA)
		require.NoError(t, err)
		require.NotNil(t, target)
		assert.Equal(t, urlA, target.URL)

		// A url that is not stored is not an error.
		target, err = s.GetUrlsByKey("http://127.0.0.1:9000/not-exist")
		require.NoError(t, err)
		assert.Nil(t, target)

		require.NoError(t, s.DeleteUrlsByKey(urlA))
		require.NoError(t, s.DeleteUrlsByKey("http://127.0.0.1:9000/not-exist"))
		target, err = s.GetUrlsByKey(urlA)
		require.NoError(t, err)
		assert.Nil(t, target)
		assert.False(t, s.IsTableEmpty())

		require.NoError(t, s.DeleteUrlsByKey(urlB))
		assert.True(t, s.IsTableEmpty())
	})

	t.Run("database errors", func(t *testing.T) {
		// The database has no target_urls table, so every query fails.
		s := &ServiceBusService{db: newTestDB(t)}

		assert.ErrorContains(t, s.InsertUrls(urlA), errNoSuchTable)
		assert.ErrorContains(t, s.DeleteUrlsByKey(urlA), errNoSuchTable)
		// A failed count is reported as an empty table.
		assert.True(t, s.IsTableEmpty())
		target, err := s.GetUrlsByKey(urlA)
		assert.ErrorContains(t, err, errNoSuchTable)
		assert.Nil(t, target)
	})
}
