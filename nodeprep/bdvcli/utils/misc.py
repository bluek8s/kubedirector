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

from ..errors import ArgumentParseError

# DIRNAME = os.path.dirname(os.path.realpath(__file__))
# SDK_DIRNAME = os.path.abspath(os.path.join(DIRNAME, '..', '..'))

def processArgs(parser, args):
    """
    Essentially performs the function of a shell wrt string processing.

    All arguments that are enclosed with in " or ' are concatinated with a space
    before handing it off to the parser for processing.
    """
    allsplit = []
    if type(args) == str:
        allsplit = args.split()
    elif  type(args) == list:
        allsplit = args
    else:
        raise Exception("Input args must be either a list or a str. (%s)" % type(args))

    retArgs = []

    appendStr = lambda x, y: ' '.join([x, y.strip('"').strip()]).strip()
    while len(allsplit) > 0:
        try:
            s = allsplit.pop(0).strip()
        except:
            break

        if s.startswith('"'):
            argString = ''
            while not s.endswith('"'):
                argString = appendStr(argString, s)
                try:
                    s = allsplit.pop(0).strip()
                except:
                    break
            argString = appendStr(argString, s)
            retArgs.append(argString)
        else:
            retArgs.append(s)

    try:
        return parser.parse_args(retArgs)
    except SystemExit:
        raise ArgumentParseError("Parsing arguments failed.")
