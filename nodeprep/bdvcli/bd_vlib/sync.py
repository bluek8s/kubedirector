#!/bin/env python
#
# Copyright (c) 2015 BlueData Software, Inc.

from fcntl import flock, LOCK_EX, LOCK_UN, LOCK_NB
from .configmeta import BDVLIB_ConfigMetadata
from .errors import UnknownInputTypeException, WakeWaitTimeoutException
from .utils import print_error, notify_wait, notify_token_wake
from time import time as timestamp
from time import sleep as sleep
from math import ceil as ceil
import os

CONFIG_SYNC_SLEEP_SEC=10
CONFIG_SYNC_DEFAULT_TIMEOUT_SEC=3600 # 1Hr timeout.

CMD_WAKE_ROLE='wake'
CMD_WAIT_FOR_ROLE='wait' # No need to poke bd_vagent to handle this CMD.

FQDN_KEY='fqdns'
GLOBAL_ID_KEY='global_id'

def BDVLIB_ServiceWait(listOfServiceKeys, timeoutSec=CONFIG_SYNC_DEFAULT_TIMEOUT_SEC):
    """
    A distributed synchronization mechanism to wait for a service that is
    expected to be available on a remote node (or on the same node).

    This call will block until all the nodes being waited on respond with a
    successful service registration or the timeout is reached.
    """
    if isinstance(listOfServiceKeys, list):
        srvcList = listOfServiceKeys
    elif isinstance(listOfServiceKeys, str):
        srvcList = listOfServiceKeys.split('.')
    else:
        print_error("Unknown input type: %s" %(type(listOfServiceKeys)))
        raise UnknownInputTypeException()

    returnStatus = 0
    timeoutMS = timeoutSec * 1000
    config = BDVLIB_ConfigMetadata()
    for key in srvcList:
        listOfKeyTokens = config.searchForToken(key, GLOBAL_ID_KEY)
        for keyTokens in listOfKeyTokens:
            globalId = config.getWithTokens(keyTokens)

            # We also want to get the FQDN where this service will be
            # configured so we can contact the remote node.
            keyTokens.pop(-1)
            keyTokens.append(FQDN_KEY)
            fqdn = ','.join(config.getWithTokens(keyTokens))

            status = notify_wait(globalId, fqdn, timeoutMS)
            if status != 0:
                returnStatus = status

    return returnStatus

def BDVLIB_TokenWait(tokenID, fqdn,
    timeoutSec=CONFIG_SYNC_DEFAULT_TIMEOUT_SEC):
    timeoutMS = timeoutSec * 1000
    status = notify_wait(tokenID, fqdn, timeoutMS)
    return status

def BDVLIB_TokenWake(tokenID, status='ok'):
    status = notify_token_wake(tokenID, status)
    return status
