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

from .configmeta import BDVLIB_ConfigMetadata
from .utils import designate_node
from .errors import DesignateTypeUndefined, DesignateMissingArgs
from .errors import UnknownConfigTypeException

BDVLIB_DESIGNATE_PRIMARY='designate_primary'
BDVLIB_DESIGNATE_SECONDARY='designate_secondary'

def BDVLIB_Designate(type, fqdn=None, config=None):
    """
    Designate a particular role to the give FQDN.

    Prameters:
        'type': BDVLIB_DESIGNATE_PRIMARY or BDVLIB_DESIGNATE_SECONDARY
        'fqdn': Fully qualified domain name of the host being designated
                as orchestrator. If none is provided the FQDN of the current
                node is used (which may not be the right thing in all cases).
        'config': A instance of BDVLIB_ConfigMetadata class. If not provided,
                  an instance is automatically created.
    Returns:
        0 on success.
        non-zero on any failure.

    Exceptions:
        DesignateTypeUndefined: When the destination type is not one of
                                BDVLIB_DESIGNATE_PRIMARY, BDVLIB_DESIGNATE_SECONDARY,
        DesignateMissingArgs: When a required argument is missing.
        UnknownConfigTypeException : When the input config parameter is not of
                                     BDVLIB_ConfigMetadata's instance.
    """
    if type not in [BDVLIB_DESIGNATE_PRIMARY, BDVLIB_DESIGNATE_SECONDARY]:
        raise DesignateTypeUndefined()

    if config is None:
        config = BDVLIB_ConfigMetadata()
    elif not isinstance(config, BDVLIB_ConfigMetadata):
        raise UnknownConfigTypeException(config)

    fqdn = config.getWithTokens(["node", "fqdn"])

    return designate_node(fqdn, type)
