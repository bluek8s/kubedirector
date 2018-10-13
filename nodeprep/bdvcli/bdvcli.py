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
from __future__ import with_statement

import os, sys, cmd, pkgutil

from . import BDVCLI_Command

from utils import isDebug
from config import VcliConfig
from utils.log import VcliLog
from errors import ArgumentParseError
from constants import BDVCLI_VERSION

from vcli import Vcli
from baseimg import Baseimage
from namespace import Namespace

## macro depends on Namesapce so import it after it.
from macro import Macro

# # FIXME! test this code
# EXTENSIONS_PATH = '/opt/bluedata/bdvcli/extensions'
# if os.path.exists(EXTENSIONS_PATH):
#     __path__ = pkgutil.extend_path(__path__, EXTENSIONS_PATH)
#     for importer, modname, ispkg in pkgutil.walk_packages(path=__path__, prefix=__name__+'.'):
#         if ispkg:
#           __import__(modname)


__all__ = ['BDvcli']

class BDvcli(cmd.Cmd):

    def __init__(self, libmode=True):
        """
        """
        self.config = VcliConfig()
        self.log = VcliLog(self.config)

        self._initialize_commands()

        self.ruler = '_'
        self.prompt = 'bdvcli> '

        if not libmode:
            self.intro = "BlueData vCLI %s.\n" %(BDVCLI_VERSION)
            self.use_rawinput = True
            cmd.Cmd.__init__(self)
        else:
            self.use_rawinput = False
            cmd.Cmd.__init__(self)

    def getCommandObject(self, name):
        """
        Returns the command object corresponding to the name, if it exists

        This is useful when bdvcli is being used as a python library to get
        the command objects. Each command object has its own publicly available
        methods.

        For example:
            namespace = bdvcli.getCommandObject('namespace')

            roleId = namespace.getValue('node.role_id')
            distroId = namesapce.getValue('node.distro_id.')
        """
        if (name != None) and (self.commands.has_key(name)):
            return self.commands[name]
        else:
            return None

    def _add_command(self, cmd, cmdObj):
        """
        Invoked by the BDVCLI_Command base class when the implementation is
        instantiated.
        """
        self.commands[cmd] = cmdObj
        cmdObj.setLogger(self.log)
        cmdObj.setConfig(self.config)

        setattr(self, 'do_' + cmd, lambda x: self.command_do(cmd, x))
        setattr(self, 'help_' + cmd, lambda : self.command_help(cmd))

    def is_interactive(self):
        """
        Check whether BVCLI is being executed intractively.
        """
        return self.use_rawinput

    def _initialize_commands(self):
        """
        Find all implementations of the base class and initialize them. This
        will allow us to extend (i.e add more functinality) by just placing the
        new implementations in the PYTHON_PATH - kind of like plugin discovery.
        """
        self.commands = {}

        for bdvcliCmd in BDVCLI_Command.__subclasses__():
            bdvcliCmd(self)

        return

    def emptyline(self):
        """
        Override this method so we don't automatically execute the previous cmd.
        """
        # do nothing.
        return


    # def onecmd(self, line):
    #     """
    #     Override the parent method to process any exceptions appropriately.
    #     """
    #     try:
    #         return cmd.Cmd.onecmd(self, arg)
    #     except Exception as e:
    #         if (self.use_rawinput == True):
    #             flush(sys.stdout)
    #             pass

    def precmd(self, line):
        """
        This overridden method added for convienience when the user executes
        help. Normally for a subcommand help, the user execute the following:

            bdvcli> vcli version -h

        instead, this override lets them execute the following:

            bdvcli> help vcli version

        The second version is more intuitive but, the first one will still work.
        """
        splits = line.split()

        if (len(splits) > 1) and splits[0] == 'help':
            if not (splits[1] == 'EOF' or splits[1] == 'exit'):
                return ' '.join(splits[1:] + ['-h'])

        finalCmdline = ' '.join(splits)
        if isDebug():
            print("EXECUTING:", finalCmdline)

        return finalCmdline

    def postcmd(self, cont, line):
        """
        This overridden method will fudge the return value of the command we
        just executed. It's normal for various commands to return None but the
        parent class treats that as a failure - which is not what we want.
        """
        if (cont == None) and (self.use_rawinput == True):
            return False

        return cont


    def command_do(self, cmd, line):
        """
        All do_<cmd> invocation by the cmd.Cmd class end up here as we added a
        method attribute to do so when instantiating the BDVCLI_Command's
        implementations (or subclasses).

        Note that the command of the form 'vcli -h' and 'vcli version -h' also
        end up here and do not get redirected to the corresponding
        help_<cmd> method.
        """
        try:
            result = self.commands[cmd].run(line)
        except ArgumentParseError as ape:
            result = None
            if not self.is_interactive():
                raise ape

        if self.is_interactive():
            print(result)
            if (result == None):
                # Keep the command loop going in interactive mode.
                return False
        else:
            return result

    def command_help(self, cmd):
        """
        This is just a dummy method so that we can add a help_<cmd> kind of
        attribute. The parent class uses that attribute to display all the
        available commands.
        """
        return

    ##############################################################
    #       default complete function                            #
    ##############################################################
    def completedefault(self, *ignored):
        (text, line, begidx, endidx) = ignored
        command = line.strip().split()[0]
        if self.commands.has_key(command):
            return self.commands[command].complete(text, line, begidx, endidx)
        else:
            return cmd.Cmd.completedefault(self, ignored)

    def get_names(self):
        """
        Override the parent's implementation to allow the dynamically added
        methods to be part of the returned list.
        """
        return dir(self)

    ##############################################################
    #                Exit the interactive shell                  #
    ##############################################################
    def do_exit(self, line):
        print("\n")
        sys.exit(0)

    def do_EOF(self, line):
        return self.do_exit(line)

    def help_exit(self):
        print('\n'.join(["exit | Ctrl+D", "\tExits the interactive shell."]))

    def help_EOF(self):
        self.help_exit()
