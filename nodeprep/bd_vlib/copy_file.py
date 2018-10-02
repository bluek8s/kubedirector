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
from .utils import copy_file

class BDVLIB_CopyFile(object):

    @classmethod
    def usage(cls):
        return "usage: bd_vcli --cp --node <node_fqdn> "\
               "--src <absolute_source_path> --dest <absolute_destination_path>"\
               " [ --perms <dest_perms_after_transfer> ]"

    def __init__(self, options):
        if  options.node == None or \
            options.src == None or \
            options.dest == None:
            raise InvalidInputException(BDVLIB_CopyFile.usage())
        self.options = options

    def run(self):
        return copy_file(
            self.options.node,
            self.options.src,
            self.options.dest,
            self.options.perms
        )
