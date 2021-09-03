# Docker Compose "Cloud Integrations"

[![Actions Status](https://github.com/docker/compose-cli/workflows/Continuous%20integration/badge.svg)](https://github.com/docker/compose-cli/actions)
[![Actions Status](https://github.com/docker/compose-cli/workflows/Windows%20CI/badge.svg)](https://github.com/docker/compose-cli/actions)

This Compose CLI tool makes it easy to run Docker containers and Docker Compose applications in the cloud using either :
- Amazon Elastic Container Service
([ECS](https://aws.amazon.com/ecs))
- Microsoft Azure Container Instances
([ACI](https://azure.microsoft.com/services/container-instances))
- Kubernetes (Work in progress)

...using the Docker commands you already know.
  
## :warning: Compose v2 (a.k.a "Local Docker Compose") has Moved

This repository is about "Cloud Integrations", the Docker Compose v2
code has moved to [github.com/docker/compose](https://github.com/docker/compose/tree/v2) 

## Getting started

To get started with Compose CLI, all you need is:

* macOS, Windows, or Windows WSL2: The current release of
  [Docker Desktop](https://www.docker.com/products/docker-desktop)
* Linux:
  [Install script](INSTALL.md)
* An [AWS](https://aws.amazon.com) or [Azure](https://azure.microsoft.com)
  account in order to use the Compose Cloud integration

Please create [issues](https://github.com/docker/compose-cli/issues) to leave feedback.

## Examples

* ECS: [Deploying Wordpress to the cloud](https://www.docker.com/blog/deploying-wordpress-to-the-cloud/)
* ACI: [Deploying a Minecraft server to the cloud](https://www.docker.com/blog/deploying-a-minecraft-docker-server-to-the-cloud/)
* ACI: [Setting Up Cloud Deployments Using Docker, Azure and Github Actions](https://www.docker.com/blog/setting-up-cloud-deployments-using-docker-azure-and-github-actions/)

## Development

See the instructions in [BUILDING.md](BUILDING.md) for how to build the CLI and
run its tests; including the end to end tests for local containers, ACI, and
ECS.
The guide also includes instructions for releasing the CLI.

Before contributing, please read the [contribution guidelines](CONTRIBUTING.md)
which includes conventions used in this project.
