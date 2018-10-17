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

from .. import ConfigCLI_SubCommand
from ..constants import BASEIMG_META_FILE

class BaseimageVersion(ConfigCLI_SubCommand):
    """

    """

    def __init__(self, cmdObj):
        ConfigCLI_SubCommand.__init__(self, cmdObj, 'version')

    def getSubcmdDescripton(self):
        return "Returns the BlueData's base image version used to build the "\
               "container image where this command is executed. Returns None "\
               "if the container image is not based on the BlueData's base image."

    def populateParserArgs(self, subparser):
        return

    def run(self, pargs):
        if os.path.exists(BASEIMG_META_FILE):
            with open(BASEIMG_META_FILE, 'r') as f:
                return f.readline().strip('\n')
        else:
            return None

    def complete(self, text, argsList):
        return []


ConfigCLI_SubCommand.register(BaseimageVersion)
__all__ = ['BaseimageVersion']
