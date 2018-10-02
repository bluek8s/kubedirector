# Copyright 2018 BlueData Software, Inc.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from .utils import print_error
from .progress import BDVLIB_Progress
from .designate import BDVLIB_Designate
from .configmeta import BDVLIB_ConfigMetadata
from .services import BDVLIB_SysVservices, BDVLIB_SystemSysVService
from .services import BDVLIB_UnregisterSysVServices, BDVLIB_UnregisterSystemSysVService
from .services import BDVLIB_SysDservices, BDVLIB_SystemSysDservices
from .services import BDVLIB_UnregisterSysDServices, BDVLIB_UnregisterSystemSysDService
from .designate import BDVLIB_DESIGNATE_PRIMARY, BDVLIB_DESIGNATE_SECONDARY
from .advconfig import BDVLIB_AdvancedConfig, BDVLIB_ADVCFG_RESTART_ALL_SRVCS
from .sync import BDVLIB_ServiceWait, BDVLIB_TokenWait, BDVLIB_TokenWake
from .copy_file import BDVLIB_CopyFile
from .exec_command import BDVLIB_ExecCommand
from .errors import *
import os

DEFAULT_BDVCLI_VERSION = '1'
RECORD_CONFIG_API_VER='/etc/guestconfig/appconfig.dat'

import os

# List of all classes/functions/constants to expose to the outside world.
__all__ = ["KeyTokenListException", "KeyTokenEmptyException",
           "UnexpectedKeyException", "UnknownValueTypeException",
           "UnknownInputTypeException", "WakeWaitTimeoutException",
           "DescTooLongException", "PercentageOutOfRangeException",
           "UnknownConfigTypeException",

           "print_error", "BDVLIB_ConfigMetadata", "BDVLIB_Progress",
           "BDVLIB_ServiceWait", "BDVLIB_Designate",
           "BDVLIB_AdvancedConfig", "startConfiguration", "BDVLIB_TokenWait",
           "BDVLIB_TokenWake", "appconfigVersionInUse",

           ## Service registration API.
           "BDVLIB_SysVservices", "BDVLIB_SystemSysVService",
           "BDVLIB_UnregisterSysVServices", "BDVLIB_UnregisterSystemSysVService",
           "BDVLIB_SysDservices", "BDVLIB_SystemSysDservices",
           "BDVLIB_UnregisterSysDServices", "BDVLIB_UnregisterSystemSysDService",

           "BDVLIB_BaseImageVersion",

           "BDVLIB_DESIGNATE_PRIMARY", "BDVLIB_DESIGNATE_SECONDARY",
           "BDVLIB_ADVCFG_RESTART_ALL_SRVCS", "DEFAULT_BDVCLI_VERSION",

           "BDVLIB_CopyFile", "BDVLIB_ExecCommand"]

def appconfigVersionInUse():
    """
    """
    if os.path.exist(RECORD_CONFIG_API_VER):
        with open(RECORD_CONFIG_API_VER, 'r') as f:
            lines = f.readlines()
            if len(lines) > 0:
                return lines[0]
            else:
                raise Exception("Unknown configuration API recorded.")
    else:
        return DEFAULT_BDVCLI_VERSION


def startConfiguration(version=DEFAULT_BDVCLI_VERSION):
    """
    Indicates the appconfig is using the specified version.
    """
    with open(RECORD_CONFIG_API_VER, 'w') as f:
        f.writelines(["%s" % (version)])

def BDVLIB_BaseImageVersion():
    """
    Get the base image version currently running.

    Return:
        - a tuple with (STR_MAJOR, STR_MINOR) version of the base image on successs.
        - IO Exception on failure.
    """
    IMG_VER_FILE='/etc/guestconfig/base_img_version'

    if os.path.exists(IMG_VER_FILE):
        with open(IMG_VER_FILE, 'r') as f:
            return tuple(f.readline().strip('\n').split('.'))
    else:
        return ("1", "0")
