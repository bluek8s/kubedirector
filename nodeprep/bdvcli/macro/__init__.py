#!/bin/env python
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

import json
import argparse

from .. import BDVCLI_Command
from ..errors import KeyError

from .node import MacroNode

class Macro(BDVCLI_Command):
    """

    """

    def __init__(self, vcli):
        BDVCLI_Command.__init__(self, vcli, 'macro',
                                'A higher order command that abstracts a group '
                                'of other commands to generate the required '
                                'information.')

        MacroNode(self)


BDVCLI_Command.register(Macro)
__all__ = ['Macro']
