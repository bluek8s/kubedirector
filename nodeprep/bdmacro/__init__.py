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

from __future__ import print_function

from bd_vlib import BDVLIB_ConfigMetadata
from abc import ABCMeta, abstractmethod
#from ..utils.misc import processArgs
import argparse, copy

class SubCommand(object):
    """

    """
    __metaclass__ = ABCMeta

    def __init__(self, subcmd = None, bdmacro = None):
        if subcmd is None:
            self.configmeta = BDVLIB_ConfigMetadata()
        else:
            # Register this SubCommand with the parent Command.
            bdmacro.addSubCommand(subcmd, self)

    def macro(self, bdmacro):
        self.bdmacro = bdmacro

    def setconfigmeta(self, configmeta):
        self.configmeta = configmeta

    @abstractmethod
    def _getSubcmdDescripton(self):
        raise Exception("Function must be implemented.")

    @abstractmethod
    def _populateParserArgs(self, subparser):
        raise Exception("Function must be implemented.")

    @abstractmethod
    def _run(self, processedArgs):
        """
        The implementation of this method must return True on successful
        completion and False on a failure.
        """
        raise Exception("Function must be implemented.")

class BDMacro(object):
    """
    """
    def __init__(self):
        self.configmeta = BDVLIB_ConfigMetadata()
        self.parser = argparse.ArgumentParser(prog='bdmacro')
        self.subparsers = self.parser.add_subparsers(dest='subcmd',
                                                     title='Subcommands')
        self.subcommands = {}

    def addSubCommand(self, cmdstr, subcmdobj):
        desc = subcmdobj._getSubcmdDescripton()
        subcmdobj.setconfigmeta(self.configmeta)
        parser_subcmd = self.subparsers.add_parser(cmdstr, help=desc,
                                                   formatter_class=argparse.ArgumentDefaultsHelpFormatter)
        subcmdobj._populateParserArgs(parser_subcmd)
        self.subcommands[cmdstr] = subcmdobj

    def help(self):
        """

        """
        self.parser.print_help()

    def dispatch(self, line):
        """
        """
        args = self.parser.parse_args(args=line)

        if args is not None:
            subcmdObj = self.subcommands[args.subcmd]
            return subcmdObj._run(args)
        else:
            self.help()
            return 1
        # else:
        #     raise Exception("Args empty")

from node import *
from exceptions import *
__all__ = ["SubCommand", "BDMacro", "BDMacroNode", "BDMacroException"]
