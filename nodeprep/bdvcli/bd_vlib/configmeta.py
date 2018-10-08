#!/bin/env python
#
# Copyright (c) 2015 BlueData Software, Inc.

import json
from .errors import *
from .utils import *
import os


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

    # def getWithTokens(self, keyTokenList):
    #     """
    #     The key input is expected to be a tokenized list. Each token in the list
    #     is used to go a level down into the metadata namespace.
    #
    #     Return Value:
    #         Returns the value corresponding to the requested key. The return
    #         data type depends on the actual value's data type:
    #             - a list as is when the value is a list.
    #             - a list of keys when the value is a dict.
    #             - a string (or unicode) when the value is a string (or unicode)
    #             - None, if the value is null.
    #
    #     Exceptions:
    #         KeyTokenListException     : the input is not a list.
    #         KeyTokenEmtpyException    : the input is an empty list.
    #         UnknownValueTypeException : the key resolved to an value but the
    #                                     data type of the value is not what was
    #                                     expected.
    #     """
    #
    #     if not isinstance(keyTokenList, list):
    #         raise KeyTokenListException("keytokens must be specified as a list "
    #                                     "instead of %s" % (type(keyTokenList)))
    #
    #     if not keyTokenList:
    #         raise KeyTokenEmptyException()
    #
    #     return _resolve_indirections(keyTokenList, self.jsonData)

    # def get(self, key, delim='.'):
    #     """
    #     Convenience implementation that always returns the value as a string or
    #     a comma delimited string.
    #
    #     The returned data is:
    #         - a comma separated list of strings when value is a list.
    #         - a comma separated list of keys when the value is a dict.
    #         - the value as is when it is a string (or unicode). This value
    #           could already be using some delimiter which is left to the caller
    #           to decipher.
    #         - an empty string, if the JSON value is a null object.
    #
    #     Exceptions:
    #         UnknownValueTypeException  : The value could not be converted to a
    #                                       comma delimited string.
    #
    #         Any exception raised by BDVCLI_ConfigMetadata.getWithTokens().
    #     """
    #     if (key == "namespaces") or (key == ''):
    #         return ','.join(["version", "node", "cluster", "distro", "services",
    #                          "tenant", "auth", "platform"])
    #
    #     data = self.getWithTokens(key.split(delim))
    #
    #     if isinstance(data, list):
    #         return ','.join(data)
    #     elif isinstance(data, bool):
    #         return "true" if data else "false"
    #     elif isinstance(data, str) or isinstance(data, unicode):
    #         return data
    #     elif isinstance(data, int):
    #         # make sure to check int AFTER bool, since bool will also match as int
    #         return str(data)
    #     elif (data is None):
    #         return ""
    #     else:
    #         raise UnknownValueTypeException("KEY: %s VALUETYPE: %s VALUE: %s" %
    #                                         (key, data, type(data)))

    # def getLocalGroupHosts(self):
    #     """
    #     Returns FQDNs of all nodes that belong to the same nodegroup as the
    #     node on which this method is invoked.
    #     """
    #     LocalNodeGrpId = self.getWithTokens([u"node", u"nodegroup_id"])
    #     return self.getNodeGroupFQDN(LocalNodeGrpId)
    #
    # def getClusterHostsFQDN(self):
    #     """
    #     Returns the FQDNs of all the nodes that are part of the cluster.
    #     """
    #     NodeGroups = self.getWithTokens([u"nodegroups"])
    #
    #     fqdnList = []
    #     for ng in NodeGroups:
    #         ret = self.getNodeGroupFQDN(ng)
    #         fqdnList.extend(ret)
    #
    #     return fqdnList
    #
    # def getNodeGroupFQDN(self, nodeGroupId):
    #     matchedKeyTokenLists = _search_token_recursive([u"nodegroups",
    #                                                     nodeGroupId],
    #                                                    u"fqdns", self.jsonData)
    #
    #     dupslist = []
    #     for keyTokenList in matchedKeyTokenLists:
    #         val = self.getWithTokens(keyTokenList)
    #
    #         if isinstance(val, list):
    #             dupslist.extend(val)
    #         else:
    #             dupslist.append(val)
    #
    #     return dict.fromkeys(dupslist).keys()

    # def getNumNodegroups(self):
    #     """
    #     Returns count of nodegroups in this cluster.
    #     """
    #     return len(self.getWithTokens([u"nodegroups"]))

    # def searchForToken(self, startKey, matchToken):
    #     """
    #     Beginning at the 'startKey' search the remainder of the name space for
    #     all occurrences of 'matchToken' and return the complete keys.
    #     """
    #     if isinstance(startKey, list):
    #         return _search_token_recursive(startKey, matchToken, self.jsonData)
    #     else:
    #         return _search_token_recursive(startKey.split('.'), matchToken,
    #                                        self.jsonData)

    # def getTenantInfoKey(self, key):
    #     """
    #     Lookup a tenant info key and return the value
    #     """
    #     return self.getWithTokens([u"cluster",
    #                                u"tenant_info",
    #                                key])
    #
    # def getTenantInfo(self):
    #     """
    #     Return all the tenant info key-value pairs
    #     """
    #     TenantInfoKeys = self.getWithTokens([u"cluster", u"tenant_info"])
    #     PropsList = []
    #     for Key in TenantInfoKeys:
    #         Value = self.getTenantInfoKey(Key)
    #         PropsList.append(str(Key) + "=" + str(Value))
    #     return PropsList
