#### MOTIVATION

Sometimes the deployer of a kdcluster will want to have a way to securely pass secret information into the containers.

There are KubeDirector features that support giving containers access to a native K8s information-containing resource such as a ConfigMap or Secret, but the kdcluster deployer may not be empowered to create such resources or to control their access privileges with enough granularity.

The "secret keys" feature addresses this situation.

The primary usecases supported in the "secret keys" design involve passing a small piece of information into a container. The canonical example here would be a decryption key, which can then be used by the application's scripts to access encrypted content from a Secret or ConfigMap.

Larger blocks of information can also be passed with this feature if you desire, but since the encrypted key is stored in the kdcluster spec it's not encouraged to pass very large blocks in this way (which will eat into the maximum allowed resource size for storing the kdcluster).

#### CONFIGURATION

To use the secret keys feature, the kdconfig resource "kd-global-config" must exist in the KubeDirector namespace. It's fine if it is created with an empty spec and allowed to populate with all defaults, but it must exist.

The "masterEncryptionKey" property in this config is related to the secret keys feature. The value of this property is a key is used to encrypt all of the values passed into the containers. Normally you will not need to worry about this value and can leave the randomly-generated default value in place.

The default value for this property is a hex-encoded 32-byte value (so therefore is 64 characters long).

If you need or want to manually specify the value, it must be a hex-encoded 16-byte, 24-byte, or 32-byte value. Those key lengths respectively correspond to the use of AES-128-GCM, AES-192-GCM, or AES-256-GCM to encrypt the secret values. You can also generate a new random 32-byte key value by deleting the existing "masterEncryptionKey" property in the config. However:

**Note:** Changing the "masterEncryptionKey" value is not currently allowed while any kdclusters exist. See [issue 512](https://github.com/bluek8s/kubedirector/issues/512) for more discussion about the reasons, and any plans toward removing this restriction.

If you are using the secret keys feature, you must keep the "masterEncryptionKey" value secure from non-administrative users. Generally you should not give any non-administrative users any privileges on the kdconfig resource type. (And often they will not even have access to the namespace where KubeDirector and the kdconfig reside.)

#### KDCLUSTER SPEC

In the "roles" list in the kdcluster spec, each list element (each role spec) can include an optional "secretKeys" list. Each element of this list has a "value" property that specifies an information string to communicate securely, and a "name" property that identifies this information. The "name" property is useful if you are passing multiple pieces of information, so that scripts inside the container can identify each piece for its intended use.

For example:
```yaml
    roles:
    - id: controller
      members: 1
      resources:
        limits:
          cpu: "1"
          memory: 2Gi
        requests:
          cpu: "1"
          memory: 2Gi
      secretKeys:
      - name: some-key-name
        value: some-key-value
      - name: some-other-key-name
        value: some-other-key-value
```

If you then read back the kdcluster spec from the K8s API, you will see that the "value" property has been removed from each of those list elements, replaced with an "encryptedValue" property. This "encryptedValue" property contains the AES-GCM encrypted form of the original "value" property. It is only for reference by KubeDirector and does not need to be directly interpreted by a user or application. The original "value" property is never visible at any time through the K8s API.

This "secretKeys" list can be specified at kdcluster creation time. When editing a kdcluster, currently you are not allowed to modify this list if the role has existing members.

In situations where editing is allowed, you can freely remove list elements, or add new elements that have "name" and "value" pairs. You are not allowed to modify the "encryptedValue" of an existing list element  -- but you could specify some new "value" instead, which would cause the "encryptedValue" to be updated.

#### ACCESS FROM THE CONTAINER

Within a container, you can use configcli tools to access the original "value" strings, for any secret-keys element specified for the role of this container.

**Note:** As with any use of configcli, this is only possible if the role in the kdapp has a "configPackage" specified.

The values are indexed by name under the "secret_keys" token, which is available for each role ID.

An example of accessing the value for the secret key named "some-key-name", using the ccli utility from the shell:
```bash
DISTRO_ID=$(ccli --get node.distro_id)
NG_ID=$(ccli --get node.nodegroup_id)
ROLE_ID=$(ccli --get node.role_id)
SECRET_KEY_NAME=some-key-name
SECRET_KEY_VALUE=$(ccli --get "distros.$DISTRO_ID.$NG_ID.roles.$ROLE_ID.secret_keys.$SECRET_KEY_NAME")
```

An example of accessing the value for the secret key named "some-key-name", using the configcli Python module:
```python
from configcli import ConfigCli
configcli = ConfigCli(shell=False)
namespace = configcli.getCommandObject("namespace")
distroID = namespace.getWithTokens(["node", "distro_id"])
ngID = namespace.getWithTokens(["node", "nodegroup_id"])
roleID = namespace.getWithTokens(["node", "role_id"])
secretKeyName = "some-key-name"
secretKeyValue = namespace.getWithTokens(["distros", distroID, ngID, "roles", roleID, "secret_keys", secretKeyName])
```

Once this value is retrieved, its use is then application-specific.

#### FUTURE WORK

* [allow secret keys to be edited while role members exist](https://github.com/bluek8s/kubedirector/issues/511)
* [better handling of masterEncryptionKey change](https://github.com/bluek8s/kubedirector/issues/512)

