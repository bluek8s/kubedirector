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

class MacroNode(BDVCLI_SubCommand):
    """
    Macros related to node
    """
    def __init__(self, vcli):
        BDVCLI_SubCommand.__init__(self, vcli, 'node')
        self.configmeta = self.vcli.getCommandObject('namespace')

    def getSubcmdDescripton(self):
        return 'Node related macros.'

    def populateParserArgs(self, subparser):
        subparser.add_argument('--get_self_index', action='store_true',
                               dest='selfindex', help='Get node index of self')
        subparser.add_argument('--get_self_id', action='store_true', dest='selfid',
                                help='Get node id of self')
        subparser.add_argument('--get_index', metavar='FQDN', action='store',
                               type=str, nargs=1, dest='nodeindex',
                               help='Get node index of self or another node, '
                               'given an fqdn argument.')
        subparser.add_argument('--get_id', metavar='FQDN', action='store',
                               type=str, nargs=1, dest='nodeid',
                               help='Get node id of self or another node, '
                               'given an fqdn argument')

    def run(self, pargs):
        if pargs.selfindex != False:
            return self.getNodeIndexSelf()
        elif pargs.selfid != False:
            return self.getNodeIdSelf()
        elif pargs.nodeindex != None:
            return self.getNodeIndexFromFqdn(pargs.nodeindex[0])
        elif pargs.nodeid != None:
            return self.getNodeIdFromFqdn(pargs.nodeid[0])
        else:
            self.parser.error("atleast one argument must be provided")

    def _prune_id_from_string(self, node_id):
        """
        Pick off the trailing node index digits and return as an int so it may be used
        as a sort criteria.
        """
        return int(node_id[len(node_id.rstrip("0123456789")):])

    def _getNodeIndexFromId(self, node_id):
        """
        Calculate node index given a node_id
        """
        all_node_id_keys = self.configmeta.searchForToken([u"nodegroups"], u"node_ids")

        node_id_list = []
        for node_id_key in all_node_id_keys:
            key = self.configmeta.getWithTokens(node_id_key)
            node_id_list.extend(key)

        # We don't know the prefix, but as we don't allow trailing digits in the prefix, we can simply
        # pluck off the trailing digits in the id and use that to locate the index of the given node_id
        # in the array. We are not allowed to import 're' for regex or this would be much simpler.

        return str(sorted(node_id_list, key=lambda name: self._prune_id_from_string(name)).index(node_id))

    def getNodeIndexSelf(self):
        """
        Return the node index of current node
        """
        node_id = self.configmeta.getWithTokens([u"node", u"id"])
        return self._getNodeIndexFromId(node_id)

    def getNodeIndexFromFqdn(self, fqdn):
        """
        Return the node index of a given fqdn
        """
        node_id = self.getNodeIdFromFqdn(fqdn)
        return self._getNodeIndexFromId(node_id)

    def getNodeIdSelf(self):
        """
        Return node id of current node
        """
        return self.configmeta.getWithTokens([u"node", u"id"])

    def getNodeIdFromFqdn(self, fqdn):
        """
        Return the node id of a given fqdn
        """
        all_fqdn_keys = self.configmeta.searchForToken([u"nodegroups"], u"fqdn_mappings")
        for fqdn_key in all_fqdn_keys:
            fqdn_token_list = self.configmeta.searchForToken(fqdn_key, fqdn)
            if fqdn_token_list != []:
                return self.configmeta.getWithTokens(fqdn_token_list[0])
        raise Exception("Failed to get node Id for the given FQDN " + str(fqdn))

    def complete(self, text, argsList):
        return []
