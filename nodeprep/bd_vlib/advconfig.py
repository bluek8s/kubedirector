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

import json
from .errors import *
from .utils import *

BDVLIB_ADVCFG_RESTART_ALL_SRVCS="all"

class BDVLIB_AdvancedConfig(object):
    """
    Parses the read-only advanced configurations stored in a file and provides
    a key-value based lookup.
    """

    def __init__(self):
        """
        """
        return

    def listNamespaces(self):
        """
        DEPRECATED: Lists the available namsepaces whose configurations are
                    specified.
        """
        return []

    def getProperties(self, Namespace):
        """
        DEPRECATED: Returns a list of 'key=value' pairs defined in the requested
                    namespace.
        """
        return []

    def restartService(self, Services=BDVLIB_ADVCFG_RESTART_ALL_SRVCS):
        """

        """
        Srvcs=''
        if isinstance(Services, list):
            Srvcs = ','.join(Services)
        elif isinstance(Services, str):
            Srvcs = Services
        else:
            raise Exception("Unknown input type: %s" % (type(Services)))

        return restart_services(Srvcs)
