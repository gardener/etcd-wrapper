# How to contribute?

Contributions are always welcome!

In order to contribute ensure that you have the development environment setup and you familiarize yourself with required steps to build, verify-quality and test.

## Setting up development environment

### Installing Go

Minimum Golang version required: `1.18`.
On MacOS run:
```bash
brew install go
```

For other OS, follow the [installation instructions](https://go.dev/doc/install).

### Installing Git

Git is used as version control for dependency-watchdog. On MacOS run:
```bash
brew install git
```
If you do not have git installed already then please follow the [installation instructions](https://git-scm.com/downloads).

### Installing Docker

In order to test dependency-watchdog containers you will need a local kubernetes setup. Easiest way is to first install Docker. This becomes a pre-requisite to setting up either a vanilla KIND/minikube cluster or a local Gardener cluster.

On MacOS run:
```bash
brew install -cash docker
```
For other OS, follow the [installation instructions](https://docs.docker.com/get-docker/).