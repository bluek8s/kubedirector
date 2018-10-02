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

import os, sys
import socket
import subprocess as sub
import getpass


def print_error(line):
    """
    Print the line to stderr.
    """
    print >> sys.stderr, str(line)

def executeCmd(cmd, shell=True):
    """
    Execute an os command and return the exit status of the command.
    """
    p = sub.Popen(cmd, stdout=sub.PIPE, stderr=sub.PIPE, shell=shell)
    out, err = p.communicate()

    status=p.returncode
    if status != 0:
        print_error("Failed to invoke '%s'. Status: %d Stdout: %s" % (cmd,
                                                                      status, out))
        if status >= 127:
            return 90
    if status == 0:
        print out
    return status

OPTION='bdvlib'
BDVAGENT='/opt/bluedata/vagent/vagent/bd_vagent/bin/bd_vagent'

def notify_wait(globalId, RemoteFqdn, timeoutMS):
    """
    Notify the remote node about this node's wait for a particular service. The
    service is uniquely identified by it's global id internally.
    """
    # FIXME! we are ignoring the timeout for now.
    execCmd = ' '.join([BDVAGENT, OPTION, 'wait', str(timeoutMS),
        globalId, RemoteFqdn])
    return executeCmd(execCmd)

def notify_token_wake(tokenId, status):
    execCmd = ' '.join([BDVAGENT, OPTION, 'wake', tokenId, status])
    return executeCmd(execCmd)

def notify_progress(progress, desc):
    """

    """
    execCmd = ' '.join([BDVAGENT, OPTION, 'progress', "%s" % progress, desc])
    return executeCmd(execCmd)

def register_service(globalId, srvcType, srvcName, srvcDesc):
    """

    """
    execCmd = ' '.join([BDVAGENT, OPTION, 'register_srvc', globalId, srvcName,
                        srvcType, srvcDesc])
    return executeCmd(execCmd)

def unregister_service(globalId):
    """

    """
    execCmd = ' '.join([BDVAGENT, OPTION, 'unregister_srvc', globalId])
    return executeCmd(execCmd)

def designate_node(fqdn, type):
    """
    """
    execCmd = ' '.join([BDVAGENT, OPTION, 'designate', type, fqdn])
    return executeCmd(execCmd)

def restart_services(srvcs):
    """
    """
    execCmd = ' '.join([BDVAGENT, OPTION, 'restart_srvc', srvcs])
    return executeCmd(execCmd)

def create_bdvcli_token():
    current_user =  getpass.getuser()
    execCmd = ' '.join([BDVAGENT, OPTION, 'create_bdvcli_token', current_user])
    return executeCmd(execCmd)

def read_bdvcli_token():
    current_user = getpass.getuser()
    token_file = "/tmp/." + current_user + "-token"
    with open(token_file, 'r') as file:
        token = file.read()
    return token

def requires_token(function):
    def wrapped(*args):
        ret_token_creation = create_bdvcli_token()
        if(ret_token_creation != 0):
            print_error("Failed to create bdvcli token")
            return ret_token_creation
        try:
            token = read_bdvcli_token()
        except:
            print_error("Failed to read bdvcli token")
            return -1
        kwargs = {"token": token}
        return function(*args, **kwargs)
    return wrapped

@requires_token
def copy_file(node, src_path, dest_path, perms, *args, **kwargs):
    """
    Copy file from src_path to dest_path@node
    """
    current_user =  getpass.getuser()
    if perms == None:
        perms = "600"
    execCmd = ' '.join([BDVAGENT, OPTION, 'copy_file', node, src_path,
                        dest_path, current_user, perms, kwargs["token"]])
    return executeCmd(execCmd)

@requires_token
def exec_command(node, command, *args, **kwargs):
    """
    Execute file at remote_path on node as user
    """
    current_user = getpass.getuser()
    execCmd = ' '.join([BDVAGENT, OPTION, 'exec_command', node, current_user,
                        command, kwargs["token"]])
    return executeCmd(execCmd)
