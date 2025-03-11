# Peerd

[![Build Status]][build-status]
[![Kind CI Status]][kind-ci-status]
[![Release CI]][release-ci]
[![CodeQL]][code-ql]
[![Go Report Card]][go-report-card]
[![codecov]][code-cov]
[![release-tag]][peerd-pkgs]

Peerd enhances [Azure Artifact Streaming] and containerd image pull performance by enabling peer-to-peer distribution in
a Kubernetes cluster. Nodes can share streamable content as well as images with each other, which can result in throughput
and latency improvements.

![cluster-ops]

## Benefits

| Benefit                                         | Artifact Streaming | Image Pulls        | Notes                                                                                                                                              |
| ----------------------------------------------- | ------------------ | ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Increased Throughput**                        | :white_check_mark: | :white_check_mark: | Both streaming and image pull latency improves.                                                                                                    |
| **Reduced Cluster Scale Out Time**              | :white_check_mark: | :white_check_mark: | New nodes stream or pull images from peers that already have that content.                                                                         |
| **Improved Cluster Fault Tolerance**            | :white_check_mark: | :white_check_mark: | Mitigating upstream throttling or unavailability.                                                                                                  |
| **Reduced Container Registry Egress Costs**     | :white_check_mark: | :white_check_mark: | By sharing content within the cluster, upstream I/O is reduced.                                                                                    |
| **More Cluster Firewall Configuration Options** | :white_check_mark: | :white_check_mark: | Once an image or streamable content is ingested, nodes can share from each other without needing to communicate with the container registry.       |
| **Ease of Use**                                 | :white_check_mark: | :white_check_mark: | Peerd is a drop-in solution that requires no changes to existing workflows or tooling, with seamless fallback to the container registry if needed. |

## Usage Guide

See the [usage guide][usage.md] to get started.

## Design and Architecture

Read the [design.md] document to understand the architecture and design of Peerd.

## Contributing

Please read our [contribution guide][CONTRIBUTING.md] which outlines all of our policies, procedures, and requirements
for contributing to this project.

## Code of Conduct

Please see [CODE_OF_CONDUCT.md] for further details.

## Acknowledgments

- Thanks to Philip Laine and Simon Gottschlag at Xenit for generously sharing their insights on [Spegel] with us.
- Thanks to [DADI P2P Proxy] for demonstrating the integration with [Overlaybd].

---

[Azure Artifact Streaming]: https://learn.microsoft.com/en-us/azure/container-registry/container-registry-artifact-streaming
[Build Status]: https://github.com/azure/peerd/actions/workflows/build.yml/badge.svg
[build-status]: https://github.com/azure/peerd/actions/workflows/build.yml
[cluster-ops]: ./assets/images//cluster-ops.gif
[codecov]: https://codecov.io/gh/Azure/peerd/branch/main/graph/badge.svg
[code-cov]: https://codecov.io/gh/Azure/peerd
[Code Coverage]: https://img.shields.io/badge/coverage-54.9%25-orange
[CODE_OF_CONDUCT.md]: CODE_OF_CONDUCT.md
[CodeQL]: https://github.com/Azure/peerd/actions/workflows/github-code-scanning/codeql/badge.svg?branch=main
[code-ql]: https://github.com/Azure/peerd/actions/workflows/github-code-scanning/codeql
[CONTRIBUTING.md]: CONTRIBUTING.md
[DADI P2P Proxy]: https://github.com/data-accelerator/dadi-p2proxy
[design.md]: ./docs/design.md
[Go Report Card]: https://goreportcard.com/badge/github.com/azure/peerd
[go-report-card]: https://goreportcard.com/report/github.com/azure/peerd
[kubectl-node-shell]: https://github.com/kvaps/kubectl-node-shell
[Kind CI Status]: https://github.com/azure/peerd/actions/workflows/kind.yml/badge.svg
[kind-ci-status]: https://github.com/azure/peerd/actions/workflows/kind.yml
[Overlaybd]: https://github.com/containerd/overlaybd
[peerd-pkgs]: https://github.com/Azure/peerd/pkgs/container/acr%2Fdev%2Fpeerd
[Release CI]: https://github.com/azure/peerd/actions/workflows/release.yml/badge.svg
[release-ci]: https://github.com/azure/peerd/actions/workflows/release.yml
[release-tag]: https://img.shields.io/github/v/tag/Azure/peerd?label=Docker%20Image%20Tag
[Spegel]: https://github.com/XenitAB/spegel
[usage.md]: ./docs/usage.md