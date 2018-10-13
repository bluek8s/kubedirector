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

from .. import BDVCLI_SubCommand

from ..errors import KeyError

class MacroNodegroup(BDVCLI_SubCommand):
    """
    Macros related to node
    """
    def __init__(self, vcli):
        BDVCLI_SubCommand.__init__(self, vcli, 'nodegroup')

    def getSubcmdDescripton(self):
        return 'Nodegroup related macros.'

    def populateParserArgs(self, subparser):
        subparser.add_argument('--get_local_group_fqdns', action='store_true',
                                dest='getLocalGroupFqdns', default=False,
                                help="Get all FQDNs deployed for the node group "
                                "that the current node belongs to.")
        subparser.add_argument('--get_nodegroup_fqdns', action='store',
                                default=None, dest='getNodeGroupFqdns',
                                help="Get all FQDNs in the given Nodegroup Id.")
        subparser.add_argument('--get_all_fqdns', action='store_true',
                                dest='getAllFqdns', default=False,
                                help='Get FQDNs of all the nodes in the cluster.')

    def run(self, pargs):

        if pargs.getLocalGroupFqdns:
            return self._get_local_nodegroup_fqdns()
        elif pargs.getNodeGroupFqdns != None:
            return self._get_nodegroup_fqdns(pargs.getNodeGroupFqdns)
        elif pargs.getAllFqdns:
            return self._get_cluster_fqdns()
        else:
            self.parser.error("atleast one argument must be provided.")

    def _get_local_nodegroup_fqdns(self):
        """
        """
        LocalNodeGrpId = self.command.configmeta.getWithTokens([u"node", u"nodegroup_id"])
        return self._get_nodegroup_fqdns(LocalNodeGrpId)

    def _get_cluster_fqdns(self):
        """
        """
        NodeGroups = self.command.configmeta.getWithTokens([u"nodegroups"])

        fqdnList = []
        for ng in NodeGroups:
            ret = self._get_nodegroup_fqdns(ng)
            fqdnList.extend(ret)

        return fqdnList

    def _get_nodegroup_fqdns(self, nodeGroupId):
        """
        """
        matchedKeyTokenLists = self.command.configmeta.searchForToken([u"nodegroups",
                                                                        nodeGroupId],
                                                                      u"fqdns")
        if len(matchedKeyTokenLists) == 0:
            raise KeyError("No nodegroup %s found." % (nodeGroupId))

        dupslist = []
        for keyTokenList in matchedKeyTokenLists:
            val = self.command.configmeta.getWithTokens(keyTokenList)

            if isinstance(val, list):
                dupslist.extend(val)
            else:
                dupslist.append(val)

        return dict.fromkeys(dupslist).keys()


    def complete(self, text, argsList):
        return []

__all__ = ["MacroNodegroup"]
