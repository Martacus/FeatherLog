﻿# GoFeather (Previously FeatherLog)
GoFeather aims to be a light-weight tool that helps you develop applications. Being a selfhosted tool this allows you to have full control over how you use GoFeather. Being open source you can also make your own changes if you so desire.

Functionalities:
- Logging
- FeatureFlags
- Authentication server.
  
# Old ReadME for FeatherLog:
FeatherLog is a light-weight logging tool that is runnable on your machine. The program utilises MongoDB for storing and retrieving logs, also allowing your other applications to connect and poll from it if you so desire.

## What it does
FeatherLog is a simple Go server application that listens for incoming log requests. When a log request is received, the server stores the log in a MongoDB database. The server also provides an endpoint for retrieving logs from the database.

## Methodology
The logging server splits logs based on a domain. The idea was that multiple localhost apps that are running need a central logging platform where logs can be viewed from. This created the idea for FeatherLog

Let's say you have two apps running, app Alpha an accounting software and app Beta an API.
You would structure the logs as follows:

```
Domain: Alpha
Group: Accounting
Tag: Error
Log: Error in the accounting software

Domain: Alpha
Group: Accounting
Tag: account_deletion
Log: Account of user ** is deleted

Domain: Beta
Group: API
Tag: CALL
Log: Call made to endpoint: **
```

This allows you to easily seperate logs by application, group and a tag. This also allows for easy filtering and searching of logs.

## Rest API
The server provides the following REST API endpoints:
```
GET /log/:domain - Retrieves all logs for the specified domain.
GET /domain/list - Retrieves a list of all domains with logs.
POST /log - Adds a new log entry to the database.
```

Post object body:
```
{
   domain    string 
   group     string  
   tag       string 
   log       string 
}
```

## Installation
You can deploy the Go server application by either using the pre-built Docker image from our container repository or by cloning the application repository, configuring your environment variables, and building your own Docker image. Below are the instructions for both methods:

### Using the Pre-Built Docker Image
Our pre-built Docker images are available in our container repository. You can pull and run the image directly, passing the necessary environment variables.

1. Pull the Docker Image
   https://github.com/users/Martacus/packages/container/package/featherlog
```sh 
docker pull ghcr.io/martacus/featherlog:latest
```

2. Use the following command to run your container, specifying your environment variables with -e flags.

```sh 
docker run -d -p 8080:8080 \
-e MONGODB_URI=your_mongodb_uri \
-e MONGODB_DB=your_database_name \
ghcr.io/martacus/featherlog:latest
``` 

### Building Your Own Docker Image
If you prefer to customize the environment or keep everything under your control, you can clone the repository, configure your environment variables in a .env file, and build your own Docker image.

#### Clone the Repository

```sh 
git clone https://github.com/Martacus/FeatherLog.git
cd FeatherLog
```

#### Configure Environment Variables

Create a new file named .env, and edit it to include your environment variable values.

```sh 
nano .env 
``` 

#### Build the Docker Image

From the root directory of the cloned repository, build your Docker image using the following command:

```sh 
docker build -t your-custom-image-name .
```

#### Run Your Docker Container

After the build completes, you can run your container with:

```sh 
docker run -d -p 8080:8080 --env-file .env your-custom-image-name
```
This will start the server on port 8080 of your host system. Adjust the port settings and environment variables according to your setup requirements.
