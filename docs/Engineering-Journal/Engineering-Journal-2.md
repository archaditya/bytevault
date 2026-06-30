# Engineering Journal #2

## Topic

Setting Up My First Production VPS (OVH Ubuntu 24.04)

## Goal

The goal was not just to rent a VPS, but to prepare a fresh Linux server so it can safely host ByteVault in the future.

This was also my first time working with a remote Linux server instead of my local machine.

### Step 1 — Connecting to the Server

After purchasing a VPS from any cloud provider they provide us the VPS IP addresses, username and temporary password.

To coonect tot eh VPS I used SSH:

```bash
ssh ubuntu@<ip>
```

**What happens here?**
SSH (Secure Shell) creates an encrypted connection between my laptop and the remote server.

Instead of physically sitting in front of the VPS, I can control it from my terminal.

---

### First Time Connection

The first connection showed

```
The authenticity of host can't be established.
```

and asked

```
Are you sure want to continue connecting?
```

I typed

```
yes
```

**What actually happened?**
Linux downloaded the server's public host key and stored it inside

```
~/.ssh/authorized_keys
```

This prevents Man-in-the-Middle attacks.

The next time I connect, SSH compares the saved fingerprint with the server's fingerprint.

If someone changes the server identity, SSH warns me.

---

### Step 2 - Login

After typing the VPS password,

I entered in my first Linux server.

```
Welcome to Ubuntu 24.04 LTS
```

I was logged in as

```
ubuntu
```

This is the default non-root user created by Cloud provider.

Using a normal user is much safer than logging in directly as root.

---

### Step 3 — Updating Ubuntu

The first thing every server should do is update itself.

Commands:

```bash
sudo apt update
```

than

```bash
sudo apt upgrade -y
```

---

**What is apt?**
APT stands for _Advanced Package Tool_.

It is Ubuntu's package manager.

Similar to

```
npm
go
pip
```

but for operating system software.

---

**Difference Between update and upgrade**
_apt update_

This does NOT install anything.

It only downloads the latest package information.

Think of it like

Refreshing the App Store before downloading apps.

---

_apt upgrade_

This actually downloads and installs the latest versions.

Without running update first,

Ubuntu doesn't even know newer versions exist.

---

**Why Sudo?**
My current user

```
ubuntu
```

does not hace permission to modify packages.

So I temporarily become the superuser using

```bash
sudo
```

which stands for

```
Super User DO
```

Only than Ubuntu allows package installation.

---

### Step 4 — Kernel Update

While upgrading,

Ubuntu showed

```
Pending kernel upgrade
```

This surprised me because even after installing updates,

the running kernel version didn't change.

---

**Why?**

The Linux kernel is the operating system itself.

It stays loaded in memory while the server is running.

Installing a newer kernel only places new files on disk.

The currently running kernel cannot replace itself.

That's why a reboot is required.

---

Command

```bash
sudo reboot
```

After reboot,

I connected again using

```bash
ssh ubuntu@<ip>
```

Now the server was running the updated kernel.

---

### Step 5 — Installing Docker

Instead of installing Docker from Ubuntu's old repository,

I installed Docker from Docker's official repository.

The process was roughly

- install required packages
- add Docker's GPG key
- add Docker repository
- update apt again
- install Docker Engine

Finally

```bash
docker --version
```

verified Docker installation.

---

**Why not use Ubuntu's Docker package?**

Ubuntu repositories are very stable,

but they are usually behind the latest Docker releases.

The official Docker repository always provides the latest stable version.

---

### Step 6 — Docker Permission Problem

Trying

```bash
docker ps
```

without sudo would normally fail because Docker communicates with

```
/var/run/docekr.sock
```

This socket belongs to

```
root
```

and the

```
docker
```

group.

Since my user wasn't in the group,

linux denied permission.

---

### Step 7 — Adding User to Docker Group

Command

```bash
sudo usermod -aG docker $USER
```

Breaking it down

```
usermod
```

Modify a user

```
-a
```

append

```
-G
```

Secondary groups

```
docker
```

Target group

```
$USER
```

Current logged-in user

---

without

```
-a
```

existing groups would have been removed,

so the append flag is extremely important.

---

### Step 8 — Refreshing Group Membership

After changing groups,

the current terminal still didn't know my user had joined the Docker group.

Instead of logging out,

I refreshed the shell.

```
newgrp docker
```

Now

```bash
groups
```

returned

```
ubuntu docker
```

meaning my user officially became part of the Docker group.

---

### Step 9 — Verification

Checking Docker

```bash
docker version
```

or

```bash
docker ps
```

worked without needing

```bash
sudo
```

which confirmed the permission setup was successful.

---

**Things I Learned**

- What SSH actually does
- Host fingerprint verification
- Why Linux uses normal users instead of root
- What `sudo` really means
- Difference between `apt update` and `apt upgrade`
- What the Linux kernel is
- Why kernel updates require reboot
- Why Docker uses a daemon
- What `/var/run/docker.sock` is
- Linux users and groups
- Why Docker creates its own group
- Meaning of `usermod -aG`
- Why `newgrp docker` is required

---

**Commands Learned**

```bash
ssh ubuntu@<ip>

sudo apt update

sudo apt upgrade -y

sudo reboot

docker --version

sudo usermod -aG docker $USER

newgrp docker

groups

docker ps
```

---

**What's Next**

- Clone ByteVault repository
- Configure Git authentication
- Install Docker Compose
- Deploy PostgreSQL and Backend using Docker Compose
- Configure Nginx as a reverse proxy
- Connect a domain and enable HTTPS with Let's Encrypt
