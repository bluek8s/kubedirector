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

// StatusGens provides thread safe access to a map of StatusGen's.
type StatusGens struct {
	lock       sync.RWMutex
	statusGens map[types.UID]StatusGen
}

// NewStatusGens is a StatusGens constructor
func NewStatusGens() *StatusGens {
	return &StatusGens{
		statusGens: make(map[types.UID]StatusGen),
	}
}

// ReadStatusGen provides thread safe read of a status gen UID string and
// validated flag.
func (s *StatusGens) ReadStatusGen(uid types.UID) (StatusGen, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	val, ok := s.statusGens[uid]
	return val, ok
}

// WriteStatusGen provides thread safe write of a status gen UID string.
// The validated flag will begin as false.
func (s *StatusGens) WriteStatusGen(uid types.UID, newGenUID string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.statusGens[uid] = StatusGen{UID: newGenUID}
}

// ValidateStatusGen provides thread safe mark-validated of a status gen.
func (s *StatusGens) ValidateStatusGen(uid types.UID) {
	s.lock.Lock()
	defer s.lock.Unlock()
	val, ok := s.statusGens[uid]
	if ok {
		val.Validated = true
		s.statusGens[uid] = val
	}
}

// DeleteStatusGen provides thread safe delete of a status gen.
func (s *StatusGens) DeleteStatusGen(uid types.UID) {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.statusGens, uid)
}

// StatusGenCount provides thread safe number of current status gens.
func (s *StatusGens) StatusGenCount() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return len(s.statusGens)
}
