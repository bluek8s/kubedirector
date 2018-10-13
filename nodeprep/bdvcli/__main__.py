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

from bdvcli import BDvcli as BDvcli
import sys, argparse


DEFAULT_TIMEOUT_SEC = 3600 # 1 hr. Allow time for a sizable cluster deployment.

def main():
    parser = argparse.ArgumentParser(description='Available options')
    parser.add_argument('instruction', nargs=argparse.REMAINDER, action='store',
                        default=None,
                        help='A single instruction to execute.')
    parser.add_argument('-v', '--version', action='store_true', dest='version',
                        help='Prints bdvcli version and exits.')

    compatArgs = parser.add_argument_group("Backward compatible options",
                        "These command line options are provided to preserve "
                        "backward compatibility.")
    compatArgs.add_argument('--baseimg_version', action='store_true',
                            help="Returns the base image version used to"
                            "build this application image.")
    compatArgs.add_argument('-g', '--get', action='store', type=str,
                            metavar='KEY', dest='namespace',
                            help="Parse the namespace key delimited by a "
                            "dot ('.') and return its value.")

    ## Distributed synchronization related arguments.
    compatArgs.add_argument('-w', '--wait', action='store', metavar='KEY',
                            type=str, dest='wait', default=None,
                            help="A dot delimited namespace KEY that specifies "
                            "services to wait for. The KEY may either identify "
                            "a specific service or a group of services but not "
                            "both at the same time.")
    compatArgs.add_argument('--tokenwait', action='store', type=str,
                            dest='token_wait', metavar='TOKEN', default=None,
                            help="A token to wait for on the specified node. "
                            "Will unblock when an explicit wake is called.")
    compatArgs.add_argument('--fqdn', action='store', type=str, dest='fqdn',
                            metavar='FQDN',
                            help="The fully qualified domain name of the host "
                            "on which the corresponding wake is expected")
    compatArgs.add_argument('-t', '--timeout', action='store', type=int,
                            dest='timeout', metavar='SECONDS', default=DEFAULT_TIMEOUT_SEC,
                            help="Maximum time (specified in seconds) to wait for "
                            "a response when -w/--wait API is invoked.")
    compatArgs.add_argument('--tokenwake', action='store',type=str,
                            dest='token_wake', metavar='TOKEN', default=None,
                            help="Wake up all cluster processes waiting on the "
                            "given token")
    compatArgs.add_argument('--success', action='store_true', dest='wake_success',
                            help="Indicates that waiters would be woken "
                            "with success status. This is the default if nothing "
                            "is specified")
    compatArgs.add_argument('--error', action='store_true',dest='wake_error',
                            help="Indicates that waiters would "
                            "be woken with error status")

    ## Application service un/registration
    compatArgs.add_argument('--service_key', action='store', type=str,
                            dest="srvc_key", metavar="KEY", default=None,
                            help="A key to uniquely identify an application's "
                            "service being registered.")
    compatArgs.add_argument('--systemv', action='store', type=str,
                            dest="sysv_service", metavar="SERVICE",
                            help="A SystemV service name who's lifecycle is to be "
                            "managed by vAgent. The \"service\" command will be used "
                            "for handling the lifecycle events.")
    compatArgs.add_argument('--systemctl', action='store', type=str,
                            dest='sysctl_service', metavar='SERVICE',
                            help="A SystemD service name who's lifecycle is to be "
                            "managed by vAgent. The 'systemctl' command will be "
                            "used for handling the lifecycle events.")
    compatArgs.add_argument('--unregister_srvc', action='store', type=str,
                            dest="unreg_app_srvc_key", metavar="KEY", default=None,
                            help="A namespace key uniquely identifying the "
                            "application's service to unregister.")

    ## File Copy API. (only works if agent is present)
    compatArgs.add_argument('--cp', action='store_true', default=None, dest='copy',
                            help="Copy file to a node. File owner (user) & "
                            "permissions can be specified")
    compatArgs.add_argument('--node', action="store", type=str,
                            help="Destination node")
    compatArgs.add_argument('--src', action="store", type=str,
                            help="Absolute path to local file")
    compatArgs.add_argument('--dest', action="store", type=str,
                            help="Absolute path to destination file")
    compatArgs.add_argument('--perms', action="store", type=str, default='600',
                            help="Permissions of the file in octal form.")

    ## Remote execute options. (only works if agent is present)
    compatArgs.add_argument('--execute', action="store_true", default=None,
                            help="Execute a file on a remote node.")
    compatArgs.add_argument('--remote_node', action="store", type=str,
                            help="Node on which to execute the file.")
    compatArgs.add_argument('--script', action="store", type=str,
                            help="Absolute path of the file to execute.")

    ## Tenant information
    compatArgs.add_argument('--tenant_info', action='store_true', dest='tenant_info',
                            help="Get the tenant specific information as "
                            "key-value pairs")
    compatArgs.add_argument('--tenant_info_lookup', action='store', type=str,
                            dest='tenant_info_key',
                            help="Lookup the value of the given key in the tenant"
                            "information.")

    ## Nodegroup level queries
    compatArgs.add_argument('--get_local_group_fqdns', action='store_true',
                            dest='getLocalGroupFqdns', default=False,
                            help="Get all FQDNs deployed for the node group that "
                            "the local node belongs to.")
    compatArgs.add_argument('--get_nodegroup_fqdns', action='store',
                            metavar='NODEGROUP_ID', dest='getNodeGroupFqdns',
                            default=None,
                            help="Get all FQDNs in the given Node group.")
    compatArgs.add_argument('--get_all_fqdns', action='store_true',
                            dest='getAllFqdns', default=False,
                            help='Get FQDNs of all the nodes in the cluster.')


    args = parser.parse_args()

    instruction = None
    if args.version == True:
        instruction = 'vcli version'
    elif args.baseimg_version == True:
        instruction = 'baseimg version'
    elif args.namespace:
        nsTokens = args.namespace.split('.')

        # Argparse will handle the case when no args are specified. So we can
        # assume we have atleast one element in the token list.
        if len(nsTokens) > 1:
            key = '.'.join(nsTokens[1:])
        else:
            key = ''

        instruction = "namespace %s %s" % (nsTokens[0], key)
    elif args.wait:
        instruction = "dsync wait --srvckey %s --timeout %d" %\
                                                    (args.wait, args.timeout)
    elif args.token_wait:
        instruction = "dsync tokenwait --token %s --fqdn %s --timeout %d" %\
                                    (args.token_wait, args.fqdn, args.timeout)
    elif args.token_wake:
        if args.success:
            tokenWakeStatus = 'success'
        elif args.error:
            tokenWakeStatus = 'error'
        else:
            # When both the options is skipped, we want to default to success.
            # The other case is just the collateral damage we can live with.
            tokenWakeStatus = 'success'

        instruction = "dsync tokenwake --token %s --status %s" %\
                                            (args.token_wake, tokenWakeStatus)
    elif args.srvc_key:
        if args.sysv_service:
            srvcTypeArg = "--systemv %s" % (args.sysv_service)
        elif args.sysctl_service:
            srvcTypeArg = "--systemctl %s" % (args.sysctl_service)
        else:
            raise argparse.ArgumentTypeError()

        instruction = 'service register --app-srvc %s %s' %\
                                        (args.srvc_key, srvcTypeArg)
    elif args.unreg_app_srvc_key:
        instruction = 'service unregister %s' %(args.unreg_app_srvc_key)
    elif args.copy:
        instruction = 'remote copy --node %s --src %s --dest %s --perms %s' %\
                                    (args.node, args.src. args.dest. args.perms)
    elif args.execute:
        instructions = 'remote execute --node %s --script %s' %\
                                                (args.remote_node, args.script)
    elif args.tenant_info:
        instruction = 'namespace tenant --info'
    elif args.tenant_info_key:
        instruction = 'namespace tenant %s' % (args.tenant_info_key)
    elif args.getLocalGroupFqdns:
        instruction = 'macro nodegroup --get_local_group_fqdns'
    elif args.getNodeGroupFqdns != None:
        instruction = 'macro nodegroup --get_nodegroup_fqdns %s' % (args.getNodeGroupFqdns)
    elif args.getAllFqdns:
        instruction = 'macro nodegroup --get_all_fqdns'
    elif len(args.instruction) > 0:
        instruction=' '.join(args.instruction)
    else:
        ## spawn the shell.
        return BDvcli(libmode=False).cmdloop()

    bdvcli = BDvcli(libmode=True)
    result = bdvcli.onecmd(instruction)
    print(bdvcli.process_result(result))

    return

if __name__ == "__main__":
    main()
