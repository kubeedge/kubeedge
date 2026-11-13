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

func TestEventBusService(t *testing.T) {
	t.Run("uses the edgecore database", func(t *testing.T) {
		s := NewEventBusService()
		require.NoError(t, s.InsertTopics("test/topic"))
		t.Cleanup(func() {
			require.NoError(t, s.DeleteTopicsByKey("test/topic"))
		})

		topics, err := s.QueryAllTopics()
		require.NoError(t, err)
		assert.Contains(t, *topics, "test/topic")
	})

	t.Run("insert, query and delete topics", func(t *testing.T) {
		s := &EventBusService{db: newTestDB(t, &models.SubTopics{})}

		_, err := s.QueryAllTopics()
		assert.EqualError(t, err, "no topics found")

		require.NoError(t, s.InsertTopics("topic/a"))
		require.NoError(t, s.InsertTopics("topic/b"))
		// Inserting an existing topic replaces it instead of adding a duplicate.
		require.NoError(t, s.InsertTopics("topic/a"))
		topics, err := s.QueryAllTopics()
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"topic/a", "topic/b"}, *topics)

		require.NoError(t, s.DeleteTopicsByKey("topic/a"))
		// Deleting a topic that does not exist is not an error.
		require.NoError(t, s.DeleteTopicsByKey("topic/not-exist"))
		topics, err = s.QueryAllTopics()
		require.NoError(t, err)
		assert.Equal(t, []string{"topic/b"}, *topics)
	})

	t.Run("database errors", func(t *testing.T) {
		// The database has no sub_topics table, so every query fails.
		s := &EventBusService{db: newTestDB(t)}

		assert.ErrorContains(t, s.InsertTopics("topic/a"), errNoSuchTable)
		assert.ErrorContains(t, s.DeleteTopicsByKey("topic/a"), errNoSuchTable)
		topics, err := s.QueryAllTopics()
		assert.ErrorContains(t, err, errNoSuchTable)
		assert.Nil(t, topics)
	})
}
