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

import os, copy
from .utils import *
from .configmeta import BDVLIB_ConfigMetadata

SERVICE_APP_SYSV='app_sysv'
SERVICE_APP_SYSD='app_sysd'
SERVICE_SYS_SYSV='sys_sysv'
SERVICE_SYS_SYSD='sys_sysd'

NAME_KEY='name'
GLOBALID_KEY='global_id'

def _get_globalid_and_name(key):
    """

    """
    if isinstance(key, str):
        keytokens = key.split('.')
    elif isinstance(key, list):
        keytokens = key

    global_keyTokenList = copy.deepcopy(keytokens)
    global_keyTokenList.append(GLOBALID_KEY)

    name_keyTokenList = copy.deepcopy(keytokens)
    name_keyTokenList.append(NAME_KEY)

    config = BDVLIB_ConfigMetadata()
    globalId = config.getWithTokens(global_keyTokenList)
    name = config.getWithTokens(name_keyTokenList)

    return (globalId, name)


def BDVLIB_SysVservices(srvc_key, srvc):
    """
    Given a list for SystemV service names this function records them so that
    the vAgent can manage it's lifecycle.

    Returns 0 when successful, otherwise a non-zero integer value.
    """
    globalId, name = _get_globalid_and_name(srvc_key)
    return register_service(globalId, SERVICE_APP_SYSV, srvc, name)

def BDVLIB_SysDservices(srvc_key, srvc):
    """
    Given a list for SystemD service names this function records them so that
    the vAgent can manage it's lifecycle.

    Returns 0 when successful, otherwise a non-zero integer value.
    """
    globalId, name = _get_globalid_and_name(srvc_key)
    return register_service(globalId, SERVICE_APP_SYSD, srvc, name)

def BDVLIB_SystemSysVService(srvc, name):
    """
    Register system service for vAgent to manage it's lifecycle. This is useful
    to register services that are not specified in the catalog entry JSON.

    Parameters:
        'srvc': The SystemV service name to manage.
        'name': Name of the service to be displayed when service stauts reports
                are generated.
    Returns:
        0 on success.
        non-zero on any failure.
    """
    return register_service(srvc, SERVICE_SYS_SYSV, srvc, name)

def BDVLIB_SystemSysDservices(srvc, name):
    """
    Register system service for vAgent to manage it's lifecycle. This is useful
    to register services that are not specified in the catalog entry JSON.

    Parameters:
        'srvc': The SystemD service name to manage.
        'name': Name of the service to be displayed when service stauts reports
                are generated.
    Returns:
        0 on success.
        non-zero on any failure.
    """
    return register_service(srvc, SERVICE_SYS_SYSD, srvc, name)

def BDVLIB_UnregisterSysVServices(srvc_key):
    """
    Unregister an application's SystemV service.

    Parameters:
        'srvc_key': The namesapce key identifying the specific service.

    Returns:
        0 on success
        non-zero on any failure
    """
    globalId, _name = _get_globalid_and_name(srvc_key)
    return unregister_service(globalId)

def BDVLIB_UnregisterSystemSysVService(srvc):
    """
    Unregister a system service.

    Parameters:
        'srvc': The name of the service used during it's registration.

    Returns:
        0 on success
        non-zero on any failure
    """
    return unregister_service(srvc)

def BDVLIB_UnregisterSysDServices(srvc_key):
    """
    Unregister an application's SystemD service.

    Parameters:
        'srvc_key': The namesapce key identifying the specific service.

    Returns:
        0 on success
        non-zero on any failure
    """
    globalId, _name = _get_globalid_and_name(srvc_key)
    return unregister_service(globalId)

def BDVLIB_UnregisterSystemSysDService(srvc):
    """
    Unregister a SystemD service.

    Parameters:
        'srvc': The name of the service used during it's registration.

    Returns:
        0 on success
        non-zero on any failure
    """
    return unregister_service(srvc)
