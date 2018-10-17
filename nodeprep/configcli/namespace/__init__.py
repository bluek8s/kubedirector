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

from .. import ConfigCLI_Command
from ..errors import KeyError
from ..constants import SECTION_ConfigCLI, KEY_CONFIGMETA_FILE, KEY_PLATFORM_INFO_FILE, KEY_PRIV_METDATA_FILE

from .node import NamespaceNode
from .version import NamespaceVersion
from .cluster import NamespaceCluster
from .distros import NamespaceDistros
from .services import NamespaceServices
from .tenant import NamespaceTenant
from .auth import NamespaceAuth
from .platform import NamespacePlatform

BDVLIB_REF_KEY_TAG='bdvlibrefkey'

class Namespace(ConfigCLI_Command):
    """

    """

    def __init__(self, ccli):
        ConfigCLI_Command.__init__(self, ccli, 'namespace',
                                'Access to all available configuration namespaces.')

        self.jsonData = None
        metaFile = self.config.get(SECTION_ConfigCLI, KEY_CONFIGMETA_FILE)
        privMetaFile = self.config.get(SECTION_ConfigCLI, KEY_PRIV_METDATA_FILE)
        platformMetaFile = self.config.get(SECTION_ConfigCLI, KEY_PLATFORM_INFO_FILE)

        try:
            with open(metaFile, 'r') as f:
                self.jsonData = json.load(f)

            # Privileged information should only be parsed for super-user.
            if (os.getuid() == 0) and os.access(privMetaFile, os.R_OK):
                with open(platformMetaFile, 'r') as f:
                    self.jsonData.update(json.load(f))

                with open(privMetaFile, 'r') as f1:
                    privJson = json.load(f1)
                    self.jsonData.update(privJson)
        except Exception:
            pass

        NamespaceNode(self)
        NamespaceVersion(self)
        NamespaceCluster(self)
        NamespaceDistros(self)
        NamespaceServices(self)
        NamespaceTenant(self)
        NamespaceAuth(self)
        NamespacePlatform(self)

    def addArgument(self, subparser):
        subparser.add_argument('key', type=str, nargs='?', default='',
                               help="A dot sperated namespace key.")


    def getValue(self, key):
        """

        """
        return self.getWithTokens(key.split('.'))

    def getWithTokens(self, keyTokens):
        """

        """
        return self._get_value(keyTokens[0], keyTokens[1:])

    def searchForToken(self, startKey, matchToken):
        """
        Beginning at the 'startKey' search the remainder of the name space for
        all occurrences of 'matchToken' and return the complete keys.
        """
        if isinstance(startKey, list):
            return self._search_token_recursive(startKey, matchToken, self.jsonData)
        else:
            return self._search_token_recursive(startKey.split('.'), matchToken,
                                                self.jsonData)

    def _get_value(self, subcmd, pargs):
        """
        Internal
        """
        keyTokens = [subcmd]
        if isinstance(pargs, argparse.Namespace):
            if pargs.key != '':
                keyTokens = keyTokens + pargs.key.split('.')
        elif isinstance(pargs, list):
            keyTokens = keyTokens + pargs
        elif isinstance(pargs, str):
            if pargs != '':
                keyTokens = keyTokens + pargs.split('.')

        try:
            return  self._resolve_indirections(keyTokens, self.jsonData)
        except KeyError as e:
            if self.ccli.is_interactive():
                return "KeyError: " + str(e)
            else:
                raise e

    def _dig_jsondata_recursive(self, keyTokenList, nextLevelJsonData):
        """
        Recursively walk down the dict of dicts until there are no more tokens in
        the requested key tokens list or a key token in the list is not found as
        a dict key in the next value.
        """
        if not keyTokenList:
            # No more key tokens to lookup so, what ever json data we have so
            # far is all we can figure out. Leave it up to the caller to figure
            # out what to do with that data.
            return (nextLevelJsonData, keyTokenList)
        else:
            try:
                currToken = keyTokenList[0]

                if isinstance(nextLevelJsonData, dict) and                     \
                    (currToken in nextLevelJsonData):
                    # Recurse to the next level to find the next key token.
                    currData = nextLevelJsonData[currToken]
                    return self._dig_jsondata_recursive(keyTokenList[1:], currData)
                else:
                    # The next token we are looking for is not available. This
                    # is a valid case if we encountered an indirect leaf. So,
                    # return the current value along with the remaining key
                    # tokens and let the caller figure out how to proceed.
                    return (nextLevelJsonData, keyTokenList)
            except Exception as e:
                # We are not expecting an exception at this point. May be the
                # key doesn't exist.
                raise UnexpectedKeyException(e)

    def _resolve_indirections(self, keyTokenList, jsonData):
        """
        The key input is expected to be a tokenized list. If a value for a
        particular key token is a dictionary and has BDVLIB_REF_KEY_TAG as a
        key, then that value is treated as an indirection and another lookup
        is performed before processing the rest of the key tokens.

        Return Value:
            Returns the value corresponding to the requested key. The return
            data type depends on the actual value's data type:
                - a list as is when the value is a list.
                - a list of keys when the value is a dict.
                - a string (or unicode) when the value is a string (or unicode)
                - None, if the value is null.

        Exceptions:
            KeyError: The input key has more tokens than what could be parsed in
                      the metadata.
            Exception: An unexpected value or exception.

        """
        try:
            data, remainingTokens = self._dig_jsondata_recursive(keyTokenList, jsonData)
        except Exception as e:
            raise KeyError("KEY: %s " % (keyTokenList), e)

        if isinstance(data, dict):
            # If the dict has a BDVLIB_REF_KEY_TAG in its keys, we are expected
            # to resolve the indirection and return the final value.
            if (BDVLIB_REF_KEY_TAG in data):
                # This is the only case where we expect some tokens to remain.
                newKeyTokens = data[BDVLIB_REF_KEY_TAG] + remainingTokens;
                return self._resolve_indirections(newKeyTokens, jsonData)
            elif not remainingTokens:
                # No more tokens left which means we found what we are looking
                # for exactly.
                return data.keys()
            else:
                # Enforce that there are no more tokens left to be parsed. We
                # may end up in this situation if the caller included the value
                # in the key token list.
                raise KeyError(keyTokenList)
        elif remainingTokens:
            # Enforce that there are no more tokens left to be parsed. We may
            # end up in this situation if the caller included the value in the
            # key token list.
            raise KeyError(remainingTokens)
        elif (data is None) or isinstance(data, list) or isinstance(data, str) or  \
             isinstance(data, unicode) or isinstance(data, bool) or isinstance(data, int):
            return data
        else:
            raise Exception("Unexpected value for KEY: %s VALUETYPE: %s VALUE: %s" %
                                            (keyTokenList, type(data), data))

    def _search_token_recursive(self, startKeyTokenList, matchKeyStr, jsonData):
        """
        Begins with the specified key, find all keys whose leaf entry matches the
        requested key. A list of all the "key token lists" is returned.
        """
        retListOf_KeyTokenLists = []
        CurrKeysLists =[startKeyTokenList]

        # Our basic algorithm is to keep replacing each element in the key token
        # list with it's sub-key token list(s).
        #
        # When processing each key token list, we determine whether or not the
        # requested match is present in it's sub-keys. If it is not we attempt
        # to go down another level, until we find what we are looking for or hit
        # a KeyTokensRemainingException. If the sub-key match is found the\
        # corresponding values are appended to the return data list.
        #
        # NOTE: We are manipulating a list (CurrKeysLists) of lists (key token list)
        while len(CurrKeysLists) > 0:
            eleList = CurrKeysLists.pop(0)

            try:
                data = self._resolve_indirections(eleList, jsonData)
            except KeyError:
                # No more subkeys for this element. And we already removed it
                # from the list so, we can skip to the next element.
                continue

            if isinstance(data, list) and (matchKeyStr not in data):
                # We only need to go to the next level if this is a list.
                # Otherwise, we reached the end of this key.
                newKeysList = []
                for s in data:
                    eleCopy = list(eleList)
                    eleCopy.append(s)
                    newKeysList.append(eleCopy)

                CurrKeysLists.extend(newKeysList)
            elif (matchKeyStr == data) or (isinstance(data, list) and          \
                                           matchKeyStr in data):
                # We have the matching leaf key we are looking for. Just process
                # that and ignore others.
                newKeyTokenList = list(eleList)
                newKeyTokenList.append(matchKeyStr)

                retListOf_KeyTokenLists.append(newKeyTokenList)

        return retListOf_KeyTokenLists


ConfigCLI_Command.register(Namespace)
__all__ = ['Namespace']
