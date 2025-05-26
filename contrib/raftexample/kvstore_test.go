// Copyright 2016 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"testing"

	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.uber.org/zap"
)

func TestKVStoreWithBackends(t *testing.T) {
	backendTypes := []BackendType{Bolt, LevelDB, Memory}

	for _, backendType := range backendTypes {
		t.Run(string(backendType), func(t *testing.T) {
			proposeC := make(chan string)
			defer close(proposeC)

			snapshotter := snap.New(zap.NewNop(), t.TempDir())
			kv := newKVStore(snapshotter, proposeC, nil, nil, backendType, "test-node")

			// Test Put and Get
			key := "test-key"
			value := "test-value"
			kv.Propose(key, value)

			// Simulate commit (simplified for testing)
			kv.mu.Lock()
			if err := kv.backend.Put(key, value); err != nil {
				t.Fatal(err)
			}
			kv.mu.Unlock()

			got, ok := kv.Lookup(key)
			if !ok || got != value {
				t.Fatalf("expected %s, got %s", value, got)
			}

			// Test Delete
			kv.Propose(key, "")
			kv.mu.Lock()
			if err := kv.backend.Delete(key); err != nil {
				t.Fatal(err)
			}
			kv.mu.Unlock()

			_, ok = kv.Lookup(key)
			if ok {
				t.Fatal("key should be deleted")
			}

			// Test Snapshot
			snapshot, err := kv.getSnapshot()
			if err != nil {
				t.Fatal(err)
			}

			if len(snapshot) == 0 {
				t.Fatal("snapshot should not be empty")
			}
		})
	}
}

func TestRecoverFromSnapshot(t *testing.T) {
	backendTypes := []BackendType{Bolt, LevelDB, Memory}

	for _, backendType := range backendTypes {
		t.Run(string(backendType), func(t *testing.T) {
			proposeC := make(chan string)
			defer close(proposeC)

			snapshotter := snap.New(zap.NewNop(), t.TempDir())
			kv := newKVStore(snapshotter, proposeC, nil, nil, backendType, "test-node")

			// Prepare test data
			data := map[string]string{"key1": "value1", "key2": "value2"}
			snapshot, err := json.Marshal(data)
			if err != nil {
				t.Fatal(err)
			}

			// Simulate recovering from snapshot
			if err := kv.recoverFromSnapshot(snapshot); err != nil {
				t.Fatal(err)
			}

			// Verify data
			for k, v := range data {
				got, ok := kv.Lookup(k)
				if !ok || got != v {
					t.Fatalf("expected %s, got %s", v, got)
				}
			}
		})
	}
}
