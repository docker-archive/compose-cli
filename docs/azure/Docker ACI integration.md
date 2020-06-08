# Run Docker containers with Azure Container Instances

Docker not only runs containers locally, but also on Azure Container Instances (ACI) using `docker run` or defined in a Compose file and deployed with `docker compose up`.

> :warning: **Note:** This is currently a beta release, details of commands and flags may change in subsequent releases.

## Log in to azure
  
You can login to Azure using the following command:
 
```console
$ docker login azure
```

This will open your Web browser and prompt you to login to Azure there.
 
## Create a context for Azure Container Instances

Once logged in, in order to deploy containers in ACI, you will need to create a Docker context that will be associated with ACI. 
You can do this with the following command: 

```console
docker context create aci myacicontext
```

The above command automatically uses your Azure login details to identify your subscription IDs and resource groups. You can then interactively select the groups that you would like to use.  
If you prefer, you can specify these options in the command line with flags `--aci-subscription-id`, `--aci-resource-group` and `--aci-location`. 

If you don't already have any existing resource groups in your Azure account, the command will create one for you, you don't need any additional options to do this.

Once you have created an ACI context, you can list your Docker contexts with the `docker context ls` command: 
```console
NAME                TYPE                DESCRIPTION                               DOCKER ENPOINT                KUBERNETES ENDPOINT   ORCHESTRATOR
myacicontext        aci                 myResourceGroupGTA@eastus                                                                     
default *           docker              Current DOCKER_HOST based configuration   unix:///var/run/docker.sock                         swarm
``` 

# Run a container

Now that you've logged in and created a context, you can then start using Docker commands to deploy some containers to ACI.  
To specify that you want to run a command using your ACI context, you can use the `--context` flag as part of your Docker command as follows: 
```console
docker --context myacicontext run -p 80:80 nginx
```

If you want all your commands to use the `myacicontext` context then you can switch the default context as follows:  
```console
$ docker context use myacicontext
$ docker run -p 80:80 nginx
```
Once you've switched to the `myacicontext` context, you can also use `docker ps` to list your containers running on ACI.

You can get logs from your container with:
 
```console
docker logs <CONTAINER_ID>
``` 

Execute a command in a running container with:

```console
docker exec -t <CONTAINER_ID>
```

To stop and remove a container from ACI, use: 

```console
docker rm <CONTAINER_ID>
```

## Running Compose applications

You can also deploy and manage multi-container applications defined in [Compose files](https://docs.docker.com/compose/compose-file/) using the `compose` command.

Using your ACI context (either with the `--context myacicontext` flag or by setting the default context with `docker context use myacicontext`), you can run `docker compose up` and `docker compose down` to start and then stop a full Compose application.

By default, the `docker compose up` command will use the `docker-compose.yaml` file in the current folder. This working directory can be specified with the `--workdir` flag or the Compose file can also be specified directly with the `--file` flag.   
You can specify a name for the Compose application using the `--name` flag when deploying it. If no name is specified, a name will be derived from the working directory.

As for single containers, you can get logs from containers that are part of the Compose application with `docker logs <CONTAINER_ID>`. (container IDs will be displayed by issuing `docker ps`).

The current Docker Azure integration does not yet allow fetching a combined log stream from all the containers making up the Compose application.

## Using ACI resource groups as namespaces

You can create several Docker contexts associated with ACI, but each with a different resource group.  
This will allow you to use these Docker contexts as namespaces. 

`docker ps` will list only containers in your current Docker context, container names or Compose application names will not collide between two different Docker contexts.   