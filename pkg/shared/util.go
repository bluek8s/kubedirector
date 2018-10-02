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
	"fmt"
	"os"
)

// StrPtr convert a string to a pointer
func StrPtr(s string) *string {
	return &s
}

// StringInList is a utility function that checks if a given string is
// present at least once in the given slice of strings.
func StringInList(
	test string,
	list []string,
) bool {

	for _, s := range list {
		if s == test {
			return true
		}
	}
	return false
}

// GetKubeDirectorNamespace is a utility function to fetch the namespace
// where kubedirector is running
func GetKubeDirectorNamespace() (string, error) {

	ns, found := os.LookupEnv(KubeDirectorNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", KubeDirectorNamespaceEnvVar)
	}
	return ns, nil
}
