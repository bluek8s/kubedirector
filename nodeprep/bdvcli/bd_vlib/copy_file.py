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
