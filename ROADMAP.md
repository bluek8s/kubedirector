# Roadmap

## Near-term

For immediate KubeDirector development plans, see the GitHub issues list, particularly the [Milestones](https://github.com/bluek8s/kubedirector/milestones).

## Towards 1.0

In longer-term plans, especially the overall effort to reach a 1.0 release, KubeDirector development has three prongs:

### Major Features

These are features that are required before we can reasonably say that KD covers the aspects of app lifecycle management that are in its purview. Current issues in this bucket:

* [live app upgrade](https://github.com/bluek8s/kubedirector/issues/229) (currently in the [on deck](https://github.com/bluek8s/kubedirector/milestone/12) milestone)
* [policy for terminated pods](https://github.com/bluek8s/kubedirector/issues/274) (currently in the [on deck](https://github.com/bluek8s/kubedirector/milestone/12) milestone)

### Ecosystem Integrations

These are interactions with non-K8s-core (but popular) services where KD would reasonably be expected to help automate integration. We'd also place in this bucket any items related to evolution of the K8s core during KD development, in ways that encourage us to rethink KD behavior.

* [Istio integrations](https://github.com/bluek8s/kubedirector/issues/484) (currently in the [0.8.0](https://github.com/bluek8s/kubedirector/milestone/13) milestone)
* [Prometheus integrations](https://github.com/bluek8s/kubedirector/issues/497) (currently in the [on deck](https://github.com/bluek8s/kubedirector/milestone/12) milestone)
* [Convergence with Application SIG work](https://github.com/bluek8s/kubedirector/issues/498) (no milestone)

### Developer Support

This is an area where we are lagging, as too much of the relevant knowledge for developing KD applications is siloed. It's not effectively tracked on the KD GitHub issues board at the moment, as the workers we do have in this area don't use that tracker. The main tasks in this area are:

* Improved CI (both for KD and for KD app regression testing)
* More example-app content in public repos
* More documentation of the KD app development process and resources
* KD app support in the Ezmeral "app workbench" from HPE
