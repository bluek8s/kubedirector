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

import json
from .errors import *
from .utils import *
import os

BDVLIB_REF_KEY_TAG='bdvlibrefkey'
PUBLIC_CONFIG_METADATA_FILE = '/etc/guestconfig/configmeta.json'
PRIV_CONFIG_METDATA_FILE = '/etc/guestconfig/.priv_configmeta.json'
PLATFORM_INFO_METADATA_FILE = '/etc/guestconfig/.platform.json'

def _dig_jsondata_recursive(keyTokenList, nextLevelJsonData):
    """
    Recursively walk down the dict of dicts until there are no more tokens in
    the requested key tokens list or a key token in the list is not found as
    a dict key in the next value.
    """
    if not keyTokenList:
        # No more key tokens to lookup so, what ever json data we have so far is
        # all we can figure out. Leave it up to the caller to figure out what to
        # do with that data.
        return (nextLevelJsonData, keyTokenList)
    else:
        try:
            currToken = keyTokenList[0]

            if isinstance(nextLevelJsonData, dict) and                         \
                (currToken in nextLevelJsonData):
                # Recurse to the next level to find the next key token.
                currData = nextLevelJsonData[currToken]
                return _dig_jsondata_recursive(keyTokenList[1:], currData)
            else:
                # The next token we are looking for is not available. This is a
                # valid case if we encountered an indirect leaf. So, return the
                # current value along with the remaining key tokens and let the
                # caller figure out how to proceed.
                return (nextLevelJsonData, keyTokenList)
        except Exception as e:
            # We are not expecting an exception at this point. May be the key
            # doesn't exist.
            raise UnexpectedKeyException(e)

def _resolve_indirections(keyTokenList, jsonData):
    """
    The key input is expected to be a tokenized list. If a value for a particular
    key token is a dictionary and has BDVLIB_REF_KEY_TAG as a key, then that
    value is treated as an indirection and another lookup is performed before
    processing the rest of the key tokens.

    Return Value:
        Returns the value corresponding to the requested key. The return
        data type depends on the actual value's data type:
            - a list as is when the value is a list.
            - a list of keys when the value is a dict.
            - a string (or unicode) when the value is a string (or unicode)
            - None, if the value is null.

    Exceptions:
        KeyTokensRemainingException: The input key has more tokens than what
                                     BD_VLIB can find in its metadata.
        UnknownValueTypeException : the key resolved to a value but the
                                    data type of the value is not what was
                                    expected.
    """
    try:
        data, remainingTokens = _dig_jsondata_recursive(keyTokenList, jsonData)
    except Exception as e:
        raise KeyLookupException("KEY: %s " % (keyTokenList), e)

    if isinstance(data, dict):
        # If the dict has a BDVLIB_REF_KEY_TAG in its keys, we are expected to
        # resolve the indirection and return the final value.
        if (BDVLIB_REF_KEY_TAG in data):
            # This is the only case where we expect some tokens to remain.
            newKeyTokens = data[BDVLIB_REF_KEY_TAG] + remainingTokens;
            return _resolve_indirections(newKeyTokens, jsonData)
        elif not remainingTokens:
            # No more tokens left which means we found what we are looking for
            # exactly.
            return data.keys()
        else:
            # Enforce that there are no more tokens left to be parsed. We may
            # end up in this situation if the caller included the value in the
            # key token list.
            raise KeyTokensRemainingException(keyTokenList)
    elif remainingTokens:
        # Enforce that there are no more tokens left to be parsed. We may
        # end up in this situation if the caller included the value in the
        # key token list.
        raise KeyTokensRemainingException()
    elif (data is None) or isinstance(data, list) or isinstance(data, str) or  \
         isinstance(data, unicode) or isinstance(data, bool) or isinstance(data, int):
        return data
    else:
        raise UnknownValueTypeException("KEY: %s VALUETYPE: %s VALUE: %s" %
                                        (keyTokenList, type(data), data))

def _search_token_recursive(startKeyTokenList, matchKeyStr, jsonData):
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
    # requested match is present in it's sub-keys. If it is not we attempt to
    # go down another level, until we find what we are looking for or hit a
    # KeyTokensRemainingException. If the sub-key match is found the corresponding
    # values are appended to the return data list.
    #
    # NOTE: We are manipulating a list (CurrKeysLists) of lists (key token list)
    while len(CurrKeysLists) > 0:
        eleList = CurrKeysLists.pop(0)

        try:
            data = _resolve_indirections(eleList, jsonData)
        except KeyTokensRemainingException:
            # No more subkeys for this element. And we already removed it from
            # the list so, we can skip to the next element.
            continue

        if isinstance(data, list) and (matchKeyStr not in data):
            # We only need to go to the next level if this is a list. Otherwise,
            # we reached the end of this key.
            newKeysList = []
            for s in data:
                eleCopy = list(eleList)
                eleCopy.append(s)
                newKeysList.append(eleCopy)

            CurrKeysLists.extend(newKeysList)
        elif (matchKeyStr == data) or (isinstance(data, list) and              \
                                       matchKeyStr in data):
            # We have the matching leaf key we are looking for. Just process
            # that and ignore others.
            newKeyTokenList = list(eleList)
            newKeyTokenList.append(matchKeyStr)

            retListOf_KeyTokenLists.append(newKeyTokenList)

    return retListOf_KeyTokenLists

class BDVLIB_ConfigMetadata(object):
    """
    Parses the read-only configuration metadata and provides a convenient
    key-value based lookup facilities.
    """
    def __init__(self, metaFile=PUBLIC_CONFIG_METADATA_FILE,
                            platformMetaFile=PLATFORM_INFO_METADATA_FILE,
                            privMetaFile=PRIV_CONFIG_METDATA_FILE):
        """
        Object initialization. Read and parse the file contents as JSON data.
        """
        with open(metaFile, 'r') as f:
            self.jsonData = json.load(f)

        # Only allow parsing the private file if invoking as super-user.
        if (os.getuid() == 0) and os.access(privMetaFile, os.R_OK):
            try:
                with open(platformMetaFile, 'r') as f:
                    self.jsonData.update(json.load(f))

                with open(privMetaFile, 'r') as f1:
                    privJson = json.load(f1)
                    self.jsonData.update(privJson)
            except Exception:
                pass

    def getWithTokens(self, keyTokenList):
        """
        The key input is expected to be a tokenized list. Each token in the list
        is used to go a level down into the metadata namespace.

        Return Value:
            Returns the value corresponding to the requested key. The return
            data type depends on the actual value's data type:
                - a list as is when the value is a list.
                - a list of keys when the value is a dict.
                - a string (or unicode) when the value is a string (or unicode)
                - None, if the value is null.

        Exceptions:
            KeyTokenListException     : the input is not a list.
            KeyTokenEmtpyException    : the input is an empty list.
            UnknownValueTypeException : the key resolved to an value but the
                                        data type of the value is not what was
                                        expected.
        """

        if not isinstance(keyTokenList, list):
            raise KeyTokenListException("keytokens must be specified as a list "
                                        "instead of %s" % (type(keyTokenList)))

        if not keyTokenList:
            raise KeyTokenEmptyException()

        return _resolve_indirections(keyTokenList, self.jsonData)

    def get(self, key, delim='.'):
        """
        Convenience implementation that always returns the value as a string or
        a comma delimited string.

        The returned data is:
            - a comma separated list of strings when value is a list.
            - a comma separated list of keys when the value is a dict.
            - the value as is when it is a string (or unicode). This value
              could already be using some delimiter which is left to the caller
              to decipher.
            - an empty string, if the JSON value is a null object.

        Exceptions:
            UnknownValueTypeException  : The value could not be converted to a
                                          comma delimited string.

            Any exception raised by BDVCLI_ConfigMetadata.getWithTokens().
        """
        if (key == "namespaces") or (key == ''):
            return ','.join(["version", "node", "cluster", "distro", "services",
                             "tenant", "auth", "platform"])

        data = self.getWithTokens(key.split(delim))

        if isinstance(data, list):
            return ','.join(data)
        elif isinstance(data, bool):
            return "true" if data else "false"
        elif isinstance(data, str) or isinstance(data, unicode):
            return data
        elif isinstance(data, int):
            # make sure to check int AFTER bool, since bool will also match as int
            return str(data)
        elif (data is None):
            return ""
        else:
            raise UnknownValueTypeException("KEY: %s VALUETYPE: %s VALUE: %s" %
                                            (key, data, type(data)))

    def getLocalGroupHosts(self):
        """
        Returns FQDNs of all nodes that belong to the same nodegroup as the
        node on which this method is invoked.
        """
        LocalNodeGrpId = self.getWithTokens([u"node", u"nodegroup_id"])
        return self.getNodeGroupFQDN(LocalNodeGrpId)

    def getClusterHostsFQDN(self):
        """
        Returns the FQDNs of all the nodes that are part of the cluster.
        """
        NodeGroups = self.getWithTokens([u"nodegroups"])

        fqdnList = []
        for ng in NodeGroups:
            ret = self.getNodeGroupFQDN(ng)
            fqdnList.extend(ret)

        return fqdnList

    def getNodeGroupFQDN(self, nodeGroupId):
        matchedKeyTokenLists = _search_token_recursive([u"nodegroups",
                                                        nodeGroupId],
                                                       u"fqdns", self.jsonData)

        dupslist = []
        for keyTokenList in matchedKeyTokenLists:
            val = self.getWithTokens(keyTokenList)

            if isinstance(val, list):
                dupslist.extend(val)
            else:
                dupslist.append(val)

        return dict.fromkeys(dupslist).keys()

    def getNumNodegroups(self):
        """
        Returns count of nodegroups in this cluster.
        """
        return len(self.getWithTokens([u"nodegroups"]))

    def searchForToken(self, startKey, matchToken):
        """
        Beginning at the 'startKey' search the remainder of the name space for
        all occurrences of 'matchToken' and return the complete keys.
        """
        if isinstance(startKey, list):
            return _search_token_recursive(startKey, matchToken, self.jsonData)
        else:
            return _search_token_recursive(startKey.split('.'), matchToken,
                                           self.jsonData)

    def getTenantInfoKey(self, key):
        """
        Lookup a tenant info key and return the value
        """
        return self.getWithTokens([u"cluster",
                                   u"tenant_info",
                                   key])
    def getTenantInfo(self):
        """
        Return all the tenant info key-value pairs
        """
        TenantInfoKeys = self.getWithTokens([u"cluster", u"tenant_info"])
        PropsList = []
        for Key in TenantInfoKeys:
            Value = self.getTenantInfoKey(Key)
            PropsList.append(str(Key) + "=" + str(Value))
        return PropsList
