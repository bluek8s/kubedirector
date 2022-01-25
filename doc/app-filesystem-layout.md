#### OVERVIEW

In the container image used for a particular role of a kdapp, generally the filesystem can have whatever layout the app requires. However there are a few KubeDirector-specific considerations that it is good to be aware of, especially if these role members can/must use persistent storage or if they have a specified application setup package.

Most of these KubeDirector behaviors are affected by a boolean flag in the kdapp resource. This is the "useNewSetupLayout" flag supported by KubeDirector v0.8.0 and later releases, which can be found in the "configPackage" object for a role (or in the top-level "defaultConfigPackage"). This flag defaults to false for backward compatibility, but you should set it to true for any new kdapp development. It's also worth considering making an update to old kdapps so that they can set this flag to true as well. The effects of this flag are covered in detail in the sections below.

#### PERSISTED DIRECTORIES

##### Concepts

Normally any changes to the container filesystem will be lost if a container needs to be restarted, for example if its hosting node goes down.

However, if you specify an amount of persistent storage for a role (when launching a kdcluster), each member in that role will get to use an associated PV of the requested size. This PV will store the content of certain filesystem directories.

Ideally these directories would ONLY contain data that is created or changed at runtime. However, in many older applications there are directories that contain a mix of immutable files (e.g. binaries), files from the container image that will be changed at runtime, and files that will be created at runtime. For these directories to have correct content on the PV, a one-time cost must be incurred at kdcluster startup, to copy all of that initial directory content over to the PV.

The set of directories-to-persist is a union of those requested by the kdapp and those required by KubeDirector itself, as described below. This includes resolving any requests that are subdirectories of other requests; for example if the app requests "/usr/local" and KubeDirector requests "/usr/local/bin", then all of "/usr/local" will be persisted on the PV.

##### Persisted on kdapp request

The kdapp resource can define directories containing data used by the application that must be persisted across container restart. This directory list is in the "persistDirs" for each kdapp role (or in the top-level "defaultPersistDirs").

The persistDirs list should specify the necessary directories-to-persist as tightly as possible. For example if you only need to persist the contents of a directory "/usr/share/foo" then that should be what you specify, as opposed to persisting all of "/usr" or "/usr/share". Casting too wide a net with the persistDirs can have a dramatic impact on kdcluster startup time when a PV is requested.

##### Always persisted

If kdcluster role requests a PV, then KubeDirector will mandate that "/etc" will always be in the list of persisted directories. This is true regardless of the kdapp configuration. Any kdapp can depend on this invariant, i.e. a kdapp does not need to specifically request persistence for "/etc" -- although doing so would be harmless.

##### Persisted if config package is used

If a role requests a PV and the kdapp defines a config package to be used in that role, then KubeDirector itself will have additional persistent directory requests.

Note that a kdapp should NOT depend on this info. If a kdapp role also requires the persistence of one of the directories mentioned below, the kdapp should explicitly request that in the role's persistDirs. This additional persistence behavior is described here only to help with kdapp development and debugging.

In the case where useNewSetupLayout is true, KubeDirector will persist these directories: "/etc", "/opt/guestconfig", "/var/log/guestconfig", "/usr/local/bin", "/usr/local/lib"

In the case where useNewSetupLayout is false, KubeDirector will persist these directories: "/etc", "/opt", "/usr"

#### CONFIG PACKAGE LOCATION

If an application config package is defined for a role, then when a member of that role first starts up the package will be installed in the member's container.

Regardless of whether the package comes from a "file://" location on the container image or is fetched by http(s), at container startup it will be extracted into "/opt/guestconfig" by KubeDirector. The exact sequence of steps (executed as the "container user") are:

```bash
cd /opt/guestconfig/
rm -rf /opt/guestconfig/*
curl -L <config package URL> -o appconfig.tgz
tar xzf appconfig.tgz
chmod +x /opt/guestconfig/*/startscript
rm -rf /opt/guestconfig/appconfig.tgz
```

The "startscript" is what will then be executed by KubeDirector as the script hook for lifecycle events.

Note that "/opt/guestconfig" is one of the locations mounted to persistent storage, if the member is using a PV. Therefore if this container is restarted, these steps (and initial configuration through the startscript) will normally be re-run only if the member is NOT using a PV.

If the member IS using a PV, these steps (and initial configuration) will be re-run only if one of the following circumstances holds true:
* The container was restarted before initial configuration could finish.
* The previous run of initial configuration ended in an error.

Since "/var/log/guestconfig" will also be persisted (if useNewSetupLayout is true), then ideally any important logging from the startscript should go to into the "/opt/guestconfig" or "/var/log/guestconfig" directory.

KubeDirector itself will log the stderr and stdout of the most recent startscript invocation to the "configure.stderr", and "configure.stdout" files in "/opt/guestconfig". The "/opt/guestconfig/configure.status" file also contains the container ID and exit status from the last run of the startscript, concatenated by an "=" character.

#### CONFIGCLI ARTIFACTS LOCATION

At any time that config package setup is going to be (re)run, KubeDirector also checks to see whether the [configcli](https://github.com/bluek8s/configcli) Python modules and scripts need to be installed in the container. The "canary" files used for this determination are "/usr/local/bin/configcli" and "/usr/bin/configcli" ... if either of those files already exist, then configcli setup is skipped.

KubeDirector does the configcli installation by injecting the configcli archive into the container's "/tmp" directory and then running the following steps (executed as the "container user"):

```bash
cd /tmp
tar xzf configcli.tgz
chmod +x /tmp/configcli-*/install
# <the configcli install script is executed at this point; see below>
rm -rf /tmp/configcli-*
rm -f /tmp/configcli.tgz
```

The configcli install script is invoked with different arguments depending on whether useNewSetupLayout is true for this member's role. The upshot in the case where useNewSetupLayout is true:
* configcli Python modules installed under "/usr/local/lib" (exact location depends on version of container's default "python" executable)
* configcli scripts and alias links ("bdvcli", "bd_vcli", "ccli", "configcli", "configmacro") installed under "/usr/local/bin"

Alternately, in the case where useNewSetupLayout is false:
* configcli Python modules installed under "/usr/lib"
* configcli scripts and alias links installed under "/usr/bin"

If useNewSetupLayout is true, then KubeDirector will configure the container so that the "PYTHONUSERBASE" environment variable is set to "/usr/local" for the "container user". Therefore when KubeDirector invokes the startscript, and startscript uses configcli, these Python modules will be loaded without issue. If for some reason your app requires that some other user account inside the container be able to load these modules, then some additional Python directories configuration may be needed for that user.

#### CONFIGCLI LEGACY SUPPORT

Application images and config packages from before KubeDirector v0.8.0 may not have "/usr/local/bin" on the PATH used when the startscript runs, and/or the scripts in the config package may have hardcoded paths to the previous "/usr/bin" locations of the configcli scripts. This has the potential to cause extra work for app developers that want to change an existing kdapp to make it work with useNewSetupLayout=true.

To take this particular issue off the table, KubeDirector creates symlinks in the old "/usr/bin" locations when useNewSetupLayout is true. Since "/usr" is not (by default) persisted when useNewSetupLayout is true, KubeDirector will re-create those symlinks if the pod container is restarted.

#### CONFIGMETA LOCATION

The "configmeta.json" file read by configcli is located in the "/etc/guestconfig" directory. If at all possible however it should not be directly parsed; access this information using the configcli scripts and Python modules.
