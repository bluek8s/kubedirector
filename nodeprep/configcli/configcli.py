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

from . import ConfigCLI_Command

from utils import isDebug
from config import CcliConfig
from utils.log import CcliLog
from errors import ArgumentParseError, UnknownValueType
from constants import ConfigCLI_VERSION

from ccli import Ccli
from baseimg import Baseimage
from namespace import Namespace

## macro depends on Namesapce so import it after it.
from macro import Macro

# # FIXME! test this code
# EXTENSIONS_PATH = '/opt/bluedata/configcli/extensions'
# if os.path.exists(EXTENSIONS_PATH):
#     __path__ = pkgutil.extend_path(__path__, EXTENSIONS_PATH)
#     for importer, modname, ispkg in pkgutil.walk_packages(path=__path__, prefix=__name__+'.'):
#         if ispkg:
#           __import__(modname)


__all__ = ['ConfigCli']

class ConfigCli(cmd.Cmd):

    def __init__(self, shell=False):
        """
        Initialize the object.

        If shell=True, the the object is intialized in an interactive mode.
        Unless launching from a shell, setting this option to True is not useful.

        To import and initialize this class in another python module, leave the
        shell value set to False.
        """
        self.config = CcliConfig()
        self.log = CcliLog(self.config, shell)

        self._initialize_commands()

        self.ruler = '_'
        self.prompt = 'ccli> '

        if shell:
            # Interactive session.
            self.intro = "Configuration CLI %s.\n" %(ConfigCLI_VERSION)
            self.use_rawinput = True
            cmd.Cmd.__init__(self)
        else:
            self.use_rawinput = False
            cmd.Cmd.__init__(self)

    def getCommandObject(self, name):
        """
        Returns the command object corresponding to the name, if it exists

        This is useful when configcli is being used as a python library to get
        the command objects. Each command object has its own publicly available
        methods.

        For example:
            namespace = configcli.getCommandObject('namespace')

            roleId = namespace.getValue('node.role_id')
            distroId = namespace.getValue('node.distro_id')
        """
        if (name != None) and (self.commands.has_key(name)):
            return self.commands[name]
        else:
            return None

    def _add_command(self, cmd, cmdObj):
        """
        Invoked by the ConfigCLI_Command base class when the implementation is
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

        for configcliCmd in ConfigCLI_Command.__subclasses__():
            configcliCmd(self)

        return

    def emptyline(self):
        """
        Override this method so we don't automatically execute the previous cmd.
        """
        # do nothing.
        return

    def precmd(self, line):
        """
        This overridden method added for convienience when the user executes
        help. Normally for a subcommand help, the user execute the following:

            configcli> ccli version -h

        instead, this override lets them execute the following:

            configcli> help ccli version

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
        method attribute to do so when instantiating the ConfigCLI_Command's
        implementations (or subclasses).

        Note that the command of the form 'ccli -h' and 'ccli version -h' also
        end up here and do not get redirected to the corresponding
        help_<cmd> method.
        """
        result = None
        try:
            res = self.commands[cmd].run(line)
            result = self.process_result(res)
        except ArgumentParseError as ae:
            # The command help will already be displayed so we don't have to
            # show the stack trace as well for this exception.
            if self.is_interactive():
                pass
            else:
                sys.exit(1)
        except Exception as e:
            if self.is_interactive():
                self.log.exception(e)
                pass
            else:
                raise e

        if self.is_interactive():
            if result != None:
                print(result)
            # keep the loop going.
            return False

        return result

    def command_help(self, cmd):
        """
        This is just a dummy method so that we can add a help_<cmd> kind of
        attribute. The parent class uses that attribute to display all the
        available commands.
        """
        return

    def process_result(self, result):
        """

        """
        if isinstance(result, list):
            return ','.join(result)
        elif isinstance(result, bool):
            return "true" if result else "false"
        elif isinstance(result, str) or isinstance(result, unicode):
            return result
        elif isinstance(result, int):
            # make sure to check int AFTER bool, since bool will also match as int
            return str(result)
        elif (result is None):
            return ""
        else:
            raise UnknownValueType("ValueType: %s Value: %s" % (type(result), result))

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
