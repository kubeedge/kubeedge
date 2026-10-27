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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	edgecorev1alpha2 "github.com/kubeedge/api/apis/componentconfig/edgecore/v1alpha2"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao"
)

// Errors returned by SQLite for queries on a database without the needed table or column.
const (
	errNoSuchTable  = "no such table"
	errNoSuchColumn = "no such column"
)

// TestMain initializes the database once for the package, because dao.Init
// only opens the database on the first call. It creates the tables of every
// module whose database service is tested in this package.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "dbclient-test")
	if err != nil {
		panic(err)
	}
	dao.Init(filepath.Join(dir, "edgecore.db"),
		&edgecorev1alpha2.MetaManager{Enable: true},
		&edgecorev1alpha2.DeviceTwin{Enable: true},
		&edgecorev1alpha2.EventBus{Enable: true},
		&edgecorev1alpha2.ServiceBus{Enable: true},
	)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// newTestDB returns a new database that only has the tables of the given models,
// so a service can be tested on its own data, and queries on other tables fail.
func newTestDB(t *testing.T, tables ...interface{}) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{
		Logger: logger.Discard,
	})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(tables...))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}
