#
# Copyright (c) 2018 BlueData Software, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from __future__ import print_function

import os

def completeFileBrowse(text, argsList):
    if len(argsList) < 2:
        path = argsList[0]
        if not os.path.isfile(path):
            dirpath = '.' if (os.path.dirname(argsList[0]) == '') else     \
                                                os.path.dirname(argsList[0])
            filename = os.path.basename(argsList[0])
            if os.path.isdir(dirpath):
                ret = [x if not os.path.isdir(os.path.join(dirpath,x)) else x + '/' \
                        for x in os.listdir(dirpath) if x.startswith(filename)]
                return ret

    return []
