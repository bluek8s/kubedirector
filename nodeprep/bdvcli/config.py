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
import ConfigParser

from .constants import *

DIRNAME = os.path.dirname(os.path.realpath(__file__))
SDK_BASE_DIR = os.path.abspath(os.path.join(DIRNAME, '..'))

class VcliConfig(object):
    """

    """

    def __init__(self):
        """
        """
        self.config = ConfigParser.SafeConfigParser(defaults={
            KEY_LOGDIR              : DEFAULT_LOG_DIR,
            KEY_PRIV_METDATA_FILE   : PRIV_CONFIG_METDATA_FILE,
            KEY_CONFIGMETA_FILE     : PUBLIC_CONFIG_METADATA_FILE,
            KEY_PLATFORM_INFO_FILE  : PLATFORM_INFO_METADATA_FILE
        })
        if os.path.exists(BDVCLI_CONFIG_FILENAME):
            self.config.read([BDVCLI_CONFIG_FILENAME])

        if not self.config.has_section(SECTION_BDVCLI):
            self.config.add_section(SECTION_BDVCLI)

    def _save(self):
        """
        Save the modified config params to the state file.
        """
        ### FIXME! Revisit the config file persistence.
        # with open(self.statefile, 'w+') as out:
        #     self.config.write(out)
        return

    def get(self, section, key, default=None):
        """

        """
        try:
            return self.config.get(section, key)
        except ConfigParser.NoSectionError, ConfigParser.NoOptionError:
            return default

    def addOrUpdate(self, section, key, value):
        """

        """
        if not self.config.has_section(section):
            self.config.add_section(section)

        self.config.set(section, key, str(value))
        self._save()

__all__ = ['VcliConfig']
