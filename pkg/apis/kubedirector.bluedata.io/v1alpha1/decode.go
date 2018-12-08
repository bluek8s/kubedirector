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

package v1alpha1

import "encoding/json"

// UnmarshalJSON for JSONSetupPackage handles the unmarshalling of three
// scenarios wrt 'setup_package':
//   1. omitted                 : IsSet==false
//   2. explicitly set to null  : IsSet==true && IsNull==true
//   3. Set to a valid object   : IsSet=true && IsNull==false
func (jsonSetupPackage *JSONSetupPackage) UnmarshalJSON(
	data []byte,
) error {

	// The fact that we entered this function means the filed is set oherwise,
	// this field will be false by default.
	jsonSetupPackage.IsSet = true

	if string(data) == "null" {
		// The field value is explicitly set to null
		jsonSetupPackage.IsNull = true
		return nil
	}

	if err := json.Unmarshal(data, &jsonSetupPackage.SetupPackage); err != nil {
		return err
	}
	jsonSetupPackage.IsNull = false

	return nil
}
