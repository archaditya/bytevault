# Learnig Docker 001

## Chapter 1 - Installing Docker (Windows)

Docker does not natively runs on the windows. Because Docker uses the linux kernel features like (namespaces, cgroups, overlay, filesystem, etc...). So to use the docker in Windos we need to install WSL2 (a lightweight linux environment) on our windows operating system

### Step 1 - WSL Check

Open PoweShell and run:

```powershell
wsl --status
```

It will give you result something like this:

```
Default Distribution: Ubuntu
Default Version: 2
```

If in place of any information you are getting an error message then you need to install WSL2.

### Step 2 - Install WSL2

1. Open PoweShell as Administrator
2. Run the following command:

```powershell
wsl --install
```

3. It will automatically install WSL2 and Ubuntu.
4. After the installation is complete, restart your computer.
   After restart Ubuntu will open and it will ask you to setup a username and password.
5. Now open PowerShell and run:

```powershell
wsl --update
```

6. It will install WSL2 kernel updates.
7. Now check the status of WSL2:

```powershell
wsl --status
```

It will give you result something like this:

```
Default Distribution: Ubuntu
Default Version: 2
```

### Step 3 - Docker Desktop

1. Go to the official docker website.
2. Download docker desktop for windows.
3. When the time of installation dont forget to check the option which says "Use WSL2 as the default

### Step 4 - First launch

When you open docker desktop first time it does these things:

```
Docker Desktop
|
WSL2
|
start Docker Engine
|
Create Docker Socket
|
Ready
```

### Verify

Poweshell or wsl2

```
docekr --version
```

than

```
docker compose version
```

than

```
docker info
```

---

# Some Observation about the docker info

## First Observation

```
OSType: linux
```

Why there is this OSType is linux even we are on the windows?

- This is the magic of Docker.

```
Windows
|
├── Docker Desktop
|
├── WSL
    |
    ├── Docker Engine (It is a software/process that run in linux )
    |
    ├── Docker Socket
```

Thts why whatever container will run inside docker will be a **Linux Container**.

## Second Observation

```
Kernel Version: 6.6.87.2-microsoft-standard-WSL2
```

This means docker does not have its own kerner, it uses the WSL2's Linux Kernel. </br></br>
That is why docker almost behaves like native Linux on windows.

## Third Observation

```
Storage Driver: overlayfs
```

This is a very important information. When will learn about the docker images and layers than will understand it much closely.

### Fourth Observation

```
Containers: 4
Running: 0
Stopped: 4
```

This means we have 4 containers in total. Out of which 0 are running and 4 are stopped. </br></br>

### Fifth Observation

```
Images: 11
```

This means we have already 11 images in total. Out of which 0 are dangling and 208 MB are cached. </br></br>

### Sixth Observation

```
Docker Root Dir

/var/lib/docker
```

_Question:_ We are on windows than form where this `/var/lib/docker` came? </br>
_Answer:_ Its not windows path its wsl's **linux filesystem** path. This is where docker keeps all its stuffs like Images, Volumes, Containers, Networks.

---
