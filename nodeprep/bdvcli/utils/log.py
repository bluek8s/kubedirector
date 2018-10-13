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

from ..constants import SECTION_BDVCLI, KEY_LOGDIR, DEFAULT_LOG_FILENAME

class VcliLog(object):
    """

    """
    def __init__(self, config):
        logDir = config.get(SECTION_BDVCLI, KEY_LOGDIR)
        logFile = os.path.join(logDir, DEFAULT_LOG_FILENAME)

        if not os.path.exists(logDir):
            os.makedirs(logDir)

        self.LOG = logging.getLogger('bdvcli')
        console_format = logging.Formatter('%(message)s')
        console_hdlr = logging.StreamHandler()
        console_hdlr.setLevel(logging.INFO)
        console_hdlr.setFormatter(console_format)
        self.LOG.addHandler(console_hdlr)

        self.LOG_FILE = logging.getLogger('bdvcli.file')
        file_formatter = logging.Formatter('%(asctime)s %(module)s %(lineno)d %(levelname)s : %(message)s')
        file_hdlr = logging.FileHandler(logFile)
        file_hdlr.setLevel(logging.DEBUG)
        file_hdlr.setFormatter(file_formatter)
        self.LOG_FILE.addHandler(file_hdlr)

        # self.LOG_CMD = logging.getLogger('bdvcli.cmd')
        # file_formatter = logging.Formatter('%(message)s')
        # file_hdlr = logging.FileHandler(logFile)
        # file_hdlr.setLevel(logging.INFO)
        # file_hdlr.setFormatter(file_formatter)
        # self.LOG_CMD.addHandler(file_hdlr)

    def debug(self, *args, **kwargs):
        self.LOG.debug(args, kwargs)

    def info(self, *args, **kwargs):
        self.LOG.info(args, kwargs)

    def warn(self, *args, **kwargs):
        self.LOG.warning(args, kwargs)

    def error(self, *args, **kwargs):
        self.LOG.error(args, kwargs)

    def exception(self, args):
        self.LOG.exception(args)

    # def instruction(self, *args, **kwargs):
    #     self.LOG_CMD.info(args, kwargs)

__all__ = ['VcliLog']
