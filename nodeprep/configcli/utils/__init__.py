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
from ..constants import ENV_ConfigCLI_DEBUG

import os
import subprocess

def printKeyVal(key,val):
    print(key,':',str(val))

def printDict(data, header=None, footer=None, indent=4):
    """
    Print the 'data' based on it's instance type with appropriate indentation.
    """
    if header != None:
        print(header)

    if isinstance(data, dict):
        for key, value in data.iteritems():
            print(indent * ' ', key, ': ', end='')
            if isinstance(value, dict):
                printDict(value, indent=indent+4)
            else:
                print(value)
    elif isinstance(data, list):
        for e in data:
            print(e, end=' ')
    elif isinstance(data, str):
        print(data)

    if footer != None:
        print(footer)

def isDebug():
    """

    """
    return os.getenv(ENV_ConfigCLI_DEBUG, 'false').lower() == 'true'


def executeShellCmd(cmd, alternateStr=None):
    """

    """
    logStr = cmd if not alternateStr else alternateStr
    if isDebug():
        print("EXECUTING: ", logStr)

    try:
        rc = subprocess.call(cmd, shell=True, stderr=subprocess.STDOUT)
        if rc != 0:

            print ("ERROR: Failed to executed command:", logStr)
            return False
    except Exception as e:
        print("Exception: ", e)
        return False

    return True
