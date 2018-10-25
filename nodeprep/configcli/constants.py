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

from version import __version__ as VERSION
import os

ConfigCLI_VERSION = VERSION

DEFAULT_BOOL_TRUE = True
DEFAULT_BOOL_FALSE = False

# BlueData internal Environment variables
ENV_ConfigCLI_DEBUG='ConfigCLI_DEBUG'

DEFAULT_LOG_DIR = "/var/log/guestconfig/configcli"
DEFAULT_LOG_FILENAME = "configcli.log"

######### Configuration files ###########
CONFIG_DIR = '/etc/guestconfig'
ConfigCLI_CONFIG_FILENAME = os.path.join(CONFIG_DIR, 'configcli.conf')
BASEIMG_META_FILE = os.path.join(CONFIG_DIR, 'base_img_version')
PUBLIC_CONFIG_METADATA_FILE = os.path.join(CONFIG_DIR, 'configmeta.json')
PRIV_CONFIG_METDATA_FILE = os.path.join(CONFIG_DIR, '.priv_configmeta.json')
PLATFORM_INFO_METADATA_FILE = os.path.join(CONFIG_DIR, '.platform.json')


######### Configuration file sections and keys ###########
# Sections
SECTION_ConfigCLI = 'configcli'

# Keys
KEY_LOGDIR = 'logdir'
KEY_CONFIGMETA_FILE = 'configmeta'
KEY_PRIV_METDATA_FILE = 'privmeta'
KEY_PLATFORM_INFO_FILE = 'platforminfo'
