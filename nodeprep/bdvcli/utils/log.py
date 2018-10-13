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

import os
import logging
import logging.handlers

from ..constants import SECTION_BDVCLI, KEY_LOGDIR, DEFAULT_LOG_FILENAME

class VcliLog(object):
    """

    """
    def __init__(self, config, interactive):
        logDir = config.get(SECTION_BDVCLI, KEY_LOGDIR)
        logFile = os.path.join(logDir, DEFAULT_LOG_FILENAME)

        if not os.path.exists(logDir):
            os.makedirs(logDir)

        self.root = logging.getLogger('bdvcli')

        if interactive:
            console_format = logging.Formatter('%(levelname)-7s: %(message)s')
            console_hdlr = logging.StreamHandler()
            console_hdlr.setLevel(logging.INFO)
            console_hdlr.setFormatter(console_format)
            self.root.addHandler(console_hdlr)

        file_formatter = logging.Formatter('%(asctime)s %(levelname)-7s: %(message)s')
        file_hdlr = logging.handlers.RotatingFileHandler(logFile,
                                                         backupCount=3,
                                                         maxBytes=1048576)
        file_hdlr.setLevel(logging.DEBUG)
        file_hdlr.setFormatter(file_formatter)
        self.root.addHandler(file_hdlr)

    def debug(self, msg, *args):
        self.root.debug(msg, *args)

    def info(self, msg, *args):
        self.root.info(msg, *args)

    def warn(self, msg, *args):
        self.root.warning(msg, *args)

    def error(self, msg, *args):
        self.root.error(msg, *args)

    def exception(self, *args):
        self.root.exception(*args)

    # def instruction(self, *args, **kwargs):
    #     self.LOG_CMD.info(args, kwargs)

__all__ = ['VcliLog']
