// Copyright 2018 BlueData Software, Inc.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shared

import (
	"k8s.io/apimachinery/pkg/types"
	"sync"
)

// StatusGen informs whether the enclosed UID has been validated.
type StatusGen struct {
	UID       string
	Validated bool
}

var (
	statusGens    map[types.UID]StatusGen
	statusGenLock sync.RWMutex
)

// ReadStatusGen provides threadsafe read of a status gen UID string and
// validated flag.
func ReadStatusGen(uid types.UID) (StatusGen, bool) {
	statusGenLock.RLock()
	defer statusGenLock.RUnlock()
	val, ok := statusGens[uid]
	return val, ok
}

// writeStatusGen provides threadsafe write of a status gen UID string.
// The validated flag will begin as false.
func WriteStatusGen(uid types.UID, newGenUID string) {
	statusGenLock.Lock()
	defer statusGenLock.Unlock()
	statusGens[uid] = StatusGen{UID: newGenUID}
}

// ValidateStatusGen provides threadsafe mark-validated of a status gen.
func ValidateStatusGen(uid types.UID) {
	statusGenLock.Lock()
	defer statusGenLock.Unlock()
	val, ok := statusGens[uid]
	if ok {
		val.Validated = true
		statusGens[uid] = val
	}
}

// deleteStatusGen provides threadsafe delete of a status gen.
func DeleteStatusGen(uid types.UID) {
	statusGenLock.Lock()
	defer statusGenLock.Unlock()
	delete(statusGens, uid)
}

func init() {
	statusGens = make(map[types.UID]StatusGen)
}
