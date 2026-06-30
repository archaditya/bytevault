# Learnig Docker 002

## Setting up a node.js project in Docker (Win-win-api)

Win-win-api is a node.js project which has three main components:

1. API
2. Worker
3. Scheduler

These all thre components will run independently in the docker.

## Docekr Setup

1. Create a `Dockerfile` in the root directory of the project.

```Dockerfile
# Use the official Node.js 22 image as the base image, This node:22 is installed from Docker Hub and it is a `complete filesystem` which has linux kernel and node.js pre-installed.
FROM node:22

# Set the working directory inside the container
WORKDIR /app

# Copy the package.json and package-lock.json files to the working directory (Host package.json -> /app/package.json)
COPY package*.json ./

# Install project dependencies
RUN npm install

# Copy the entire application source code to the working directory (This is after npm install because if any change occurs in files except package*.json the installtion process will not trigger.)
COPY . .

# Expose the port that the application listens on
EXPOSE 3000

# Command to run the application
CMD ["npm", "start"]
```

2. Command `docker build -t winwin-api:v1 .`

- This commands builds the image and tags it with `winwin-api:v1`.
- The `.` at the end of the command specifies the build context, which is the current directory.

3. `.dockerignore` file

This file tells Docker which files and directories to ignore when building the image. This is done to reduce the size of the image and to improve the build time.

```
node_modules
dist
build
.git
.gitignore
.env
.env.local
.env.development.local
.env.test.local
.env.production.local
```

4. Command `docker run -d -p 3000:3000 --name win-win-api winwin-api:v1`

- This commands runs the container in detached mode (`-d`).
- It maps the host port 3000 to the container port 3000 (`-p 3000:3000`).
- It names the container `win-win-api` (`--name win-win-api`).
- It runs the image `winwin-api:v1` (`winwin-api:v1`).

**Note** : We are currently using the host machine MySQL and Redis so instead of using localhost/127.0.0.1 will use the `host.docker.internal` as MySQL and Redis are not in docker.

As we have put the .env file in `.dockerignore` file so when we run our application, the application will not get any environment variable and it will give error. To avoid this we can use the `--env-file` flag to pass the environment variables to the container.

`docker run -d -p 3000:3000 --env-file .env winwin-api:v1`

But here is a catch this will work file if we have only one process to run but in this project we have worker and schedular also to run. So we have to run three different containers for each process.

So we will create a `docker-compose.yml` file to run all the three processes in different containers.

---
