# Docker Learning 003

## Docker Compose

### Our current situation
If we have to run API than
```bash
docker run \
  --env-file .env \
  -p 8001:8001 \
  winiwn-api:v2
```
Worker
```bash
docker run \
  --env-file .env \
  winwin-api:v2 \
  node dist/bin/worker.js
```
Scheduler
```bash
docker run \
  --env-file .env \
  winwin-api:v2 \
  node dist/bin/scheduler.js
```
And different commands for Redis and MySQL.

Now the the question is who is responsible for this? Developer or project configuration.

Obviously, Project configuration.

So, to solve this problem we use **Docker Compose**.

---

## What is Docker Compose?
Compose is Infrastructure as Code.

Like we write code for backend.

Same we write code for the infrastructure also

Instead of remembering:
```bash
docker run ...
docker run ...
docker run ...
docker network create ...
docker volume create ...
```
We just write a single file.
```
docker-compose.yml
```
then docker will manage everything.

---

## Design the Architecture
Our services
```
API
Worker
Scheduler
Redis
MySQL
```
Question.

What all services we are writing?

---
**API**
Ours

---

**Worker**
Ours

---

**Scheduler**
Ours

---

**Redis**
Docker Hub

---

**MySQL**
Docker Hub

---

In-short
```
        Services

API         ---> custom image

Worker      ---> custom image

Scheduler   ---> custom image

Redis       ---> official image

MySQL       ---> official image
```

---

### API / Worker / Scheduler
Actually, these 3 are not different applications.

These are three entry points of a single codebase.
```
win-win-api

        │

        ├── api.js

        ├── worker.js

        └── scheduler.js
```
This is why

One Image.

Three containers.

---

### First design decision
Will only build image one in the compose.

### Compose Skeleton
```YAML
services:

  api:

  worker:

  scheduler:

  redis:

  mysql:
```
This it.

---

### API Service
What is the minimum codes /commands required for a service to run?

If we look closely at `docker run`
```bash
docker run \
  --env-file .env \
  -p 8001:8001 \
  winwin-api:v2
```
These all information will convert into YAML.

So what all the information are in this `docker run`:
```
winwin-api:v2     -> image: (or build:)

--env-file .env   -> env_file:

-p 8001:8001      -> ports:
```

One important thing is missing.

As we are run
```bash
docker run winwin-api:v2
```
Than by default
```
CMD ["node", "dist/bin/api.js"]
```
Runs, which is there in the Dockerfile.

But for the worker
```bash
docker run winwin-api:v2 \
  node dist/bin/worker.js
```
what did we do here?

**We override the CMD**

This is the same thing happens in then compose:
```YAML
worker:
  command: node dist/bin/worker.js
```
And scheduler:
```YAML
scheduler:
  command: node dist/bin/scheduler.js
```

---

## Important Concept
As of now we are using image as:
```
Dockerfile
      │
      ▼
CMD = node dist/bin/api.js
```
API container:
```
Image
 └── CMD
      │
      ▼
api.js
```
Worker container:
```
Same Image
      │
      ▼
Compose overrides CMD
      │
      ▼
worker.js
```
Same goes for scheduler as well.

That is why one image can perform multiple roles.

This is a very common pattern in production.

---

Now for the API will write something like this in Compose:
```YAML
api:
  build: .
```
This will build the image.

But for the worker and scheduler we need to pull this image to use.

So this below approach is best:
```YAML
x-app: &app
  build: .
  env_file:
    - .env

services:

  api:
    <<: *app
    command: node dist/bin/api.js
    ports:
      - "8000:8000"

  worker:
    <<: *app
    command: node dist/bin/worker.js

  scheduler:
    <<: *app
    command: node dist/bin/scheduler.js
```
Now Compose will build image if image not exist.

This will make things reliable, as any one will clone this project and directly run `docker compose up` than it will build image and use that image.

---

Generally, there are two approaches in the Compose:

**Development**
```YAML
build: .
```
Reason:
- changes in code code frequently
- Image builds on local machine

---

**Production**
```YAML
image: ghcr.io/company/win-win-api:1.0.5
```
Reason:
- Image has already build in the CI/CD
- Server only pulls the image
- Production server does not need source code

## Writing docker-compose.yaml

### Step 1 - File Creation
In the project root create
```
docker-compose.yml
```

### Step 2 - Root Key
```YAML
services:
```
Each service eventually be a Container

### Step 3 - API Service
```YAML
services:
  api
```
`api` is a logical name only.

its not an image name.

or its also not a container name.

This is only a service identifier.

So that we can write:
```bash
docker compose up api
```
or
```bash
docker compose logs api
```

---

### Step 4 - Build
```YAML
services:
  api:
    build: .
```
What is this `.`?
This is current directory.

Compose initially runs this command:
```bash
docker build .
```
And the Dockerfile is in this directory also.

---

**If in case there is different name of Dockerfile**
Example:
```
docker/
  Dockerfile.dev
```

Than we write:

```YAML
build:
  context: .
  dockerfile: docker/Dockerfile.dev
```

---

### Step 5 - image
```YAML
services:
  api:
    build: .
    image: winwin-api:latest
```
image is used to provide names to image. If we does not specify the image name it autogenerate name with formats like:
```
<project-name>-<service-name>

<project-name>-api
```

---

### Step 6 - container_name
By default docker generates container names
examples:
```
happy_einstein

win-win-api-api-1
```
but if we put `container_name` in YAML than it will use that name only
```YAML
services:
  api:
    build: .
    image: winwin-api:latest
    container_name: winwin-api
```
In production environment we avoid naming the containers as we have replicas so if we fixed the name than it will create unnecessary confusion.

### Step 7 - ports
```YAML
services:
  api:
    build: .
    image: winwin-api:latest
    container_name: winwin-api
    ports:
      - "8001:8001"
```
It does mapping between host machine port and Container port

Syntax:
```
HOST_PORT:CONTAINER_PORT
```
Important:
```
LEFT = Host
RIGHT = Container
```

Visualization:
```
              Host (My Laptop)
          localhost:8001
                 │
                 │
            8001 : 8001
                 │
                 ▼
          Container (API)
            Port 8001
```

---

### Step 8 - Env
```YAML
services:
  api:
    build: .
    image: winwin-api:latest
    container_name: winwin-api
    ports:
      - "8001:8001"
    env_file:
      - .env
```
It injects all the environment variables of `.env` inside container

Or

```YAML
environment:
  DB_HOST: mysql
  PORT: 8000
```

---

### Step 9 - Redis Service
Inside `services`, add:
```YAML
services:
  api:
    build: .
    image: winwin-api:latest
    container_name: winwin-api
    ports:
      - "8000:8000"
    env_file:
      - .env

  redis:
    image: redis:7-alpine
    container_name: winwin-redis
```

**Breakdown:**
---
**redis**:
- `redis` : This is logical name of service
- It is used in compose command: `docker compose logs redis`

---

**image: redis:7-alpine**
- This is the official redis image from Docker Hub
- If this image does not exist internally than compose will pull it from docker hub like this `docker pull redis:7-alpine`

---

**What is `7-alpine`?**
Image name:
```
redis
```
Tag:
```
7-alpine
```
- Redis Version = 7 
- Base OS = Alpine Linux

---

**`container_name`**
```YAML
container_name: winwin-redis
```
it is container name.

---

> **Note**: Official Docker Images generally run using `image:` because their maintainers already defined their Dockerfile and startup commands. So we do not need to build them again.

---

### Step 10 - Assigning `ports` to Redis
```YAML
redis:
  image: redis:7-alpine
  container_name: winwin-redis
  ports:
    - "6379:6379"
```
In the development we are assigning ports here because if **Developer** wants to access redis form host machine than he can access it.

**Internal Communication** (API -> Redis) is done using Docker network so no need of port. 

**External Communication** (Laptop -> Redis) is Host -> Container communication so we need ports for that.

That's why in production we do not write ports in the `docker-compose` for redis.

Event we are not specified any network related things till now. even though Compose will internally generate a bridge network `winwin-default`. 

And both the containers will be joined inside it:
```
                 winwin_default

      +------------------------------+

      API Container

             │

             │ redis

             ▼

      Redis Container

      +------------------------------+
```

### Step 11 - MySQL Service
```YAML
mysql:
  image: mysql:8.4
  container_name: winwin-mysql
```

> **Note**: Official images have different initialization requirements fro each one of them. Redis can start with default configurations but MySQL needs some configurations mandatorily on startup. So we need to provides some `environment` variables.

---

### Step 12 - MySQL Environments
```YAML
mysql:
  image: mysql:8.4
  container_name: winwin-mysql
  environment:
    MYSQL_ROOT_PASSWORD: root
    MYSQL_DATABASE: winwin
    MYSQL_USER: aditya
    MYSQL_PASSWORD: password
```

---

**What is this `environment` does?**

This sets the environment variables of container.

When MySQL container starts first time, MySQL startup script reads these variables:
```
MYSQL_ROOT_PASSWORD
        │
        ▼
Create root user

MYSQL_DATABASE
        │
        ▼
create Database

MYSQL_USER
        │
        ▼
create normal user

MYSQL_PASSWORD
        │
        ▼
Set user password
```

---


### Step 13 - Volumes
```YAML
mysql:
  image: mysql:8.4
  container_name: winwin-mysql
  environment:
    MYSQL_ROOT_PASSWORD: root
    MYSQL_DATABASE: winwin
    MYSQL_USER: aditya
    MYSQL_PASSWORD: password
  volumes:
    - mysql-data:/var/lib/mysql
```
And at the end of file add this:
```YAML
volumes:
  mysql-data:
```

---

**Where MySQL actually stores data?**

Inside the container:
```
MySQL Container

/
├── app
├── etc
├── usr
├── var
│   └── lib
│       └── mysql   <-- Database files
```
this `/var/lib/mysql` is the default data directory of MySQL.

This place contains all the:
- tables
- indexes
- users
- passwords
- transactions


**The Problem**

As the container is deleted, the data also gets deleted.

That is the reason we use `volumes`.
```
Container

/var/lib/mysql
        │
        │
        ▼

Docker Volume

mysql-data
```
Now database is not inside the container.

---

Look at this syntax:
```
mysql-data:/var/lib/mysql
```

**Left Side**

Docker Volume
```
mysql-data
```

**Right Side**

Container Path
```
/var/lib/mysql
```
This means, mount this Docker volume with container's `/var/lib/mysql`.

---

These volumes are generally in the Linux at location `/var/lib/docker/volumes/`.
---

> **Note:** Containers are ephemeral, but databases require persistent storage. Docker Volumes provide storage that exists independently of a container. By mounting mysql-data to /var/lib/mysql, MySQL stores its database files in a persistent Docker-managed volume instead of the container's writable layer. Deleting the container does not delete the volume or the stored data.

---

### Step 13 - MySQL Ports
```YAML
mysql:
  image: mysql:8.4
  container_name: winwin-mysql
  environment:
    MYSQL_ROOT_PASSWORD: root
    MYSQL_DATABASE: winwin
    MYSQL_USER: aditya
    MYSQL_PASSWORD: password
  ports:
    - "3306:3306"
  volumes:
    - mysql-data:/var/lib/mysql
```

--- 

> **Note**: Exposing MySQL with 3306:3306 is mainly a development convenience. It allows database tools running on the host (Workbench, DBeaver, etc.) to connect to the MySQL container. Containers inside the same Docker network do not require exposed ports to communicate with each other.


### Step 14 - `depends-on`
```YAML
api:
  build: .
  image: winwin-api:latest
  container_name: winwin-api
  ports:
    - "8000:8000"
  env_file:
    - .env
  depends_on:
    - mysql
    - redis
```
It tells the Docker Compose that before starting API container start MySQL and Redis containers.

Flow:
```
docker compose up

        │
        ▼

Start mysql

        │
        ▼

Start redis

        │
        ▼

Start api
```
--- 

Here is a catch.

It takes some time to start the MySQL.

Example Timeline:
```
0 sec

docker starts mysql container
        │
        ▼

MySQL startup script chal rahi hai
Database create ho raha hai

        │

2 sec

API container start ho gaya

        │

API tries to connect

        │

ECONNREFUSED ❌

        │

5 sec

MySQL finally ready
```
This is very common production issue.

---

To solve this we user 3 approaches:

1. Retry Logic
```
Try DB

↓

Fail

↓

Wait 2 sec

↓

Try again

↓

Success
```

2. Healthcheck + depends_on
We write Healthcheck:
```
MySQL Healthy

↓

Than API start
```

3. Wait-for-it.sh
Very old approach.

---

> **Note**: `depends_on` only controls the startup order of containers. It does not guarantee that a dependency is fully initialized or ready to accept connections. Production applications typically combine `depends_on` with health checks or implement retry logic during startup.