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

from .errors import InvalidInputException
from .utils import exec_command


class BDVLIB_ExecCommand(object):

    @classmethod
    def usage(cls):
        return "usage: bd_vcli --exec --remote_node <node_fqdn> "\
               "--script <absolute_path_on_remote>"

    def __init__(self, options):
        if  options.remote_node == None or \
            options.script == None:
            raise InvalidInputException(BDVLIB_ExecCommand.usage())
        self.options = options

    def run(self):
        return exec_command(
            self.options.remote_node,
            self.options.script
        )
