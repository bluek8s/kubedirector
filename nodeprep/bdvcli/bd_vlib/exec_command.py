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
