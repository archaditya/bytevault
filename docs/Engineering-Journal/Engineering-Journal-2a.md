# Engineering Journal #2A

## Topic

Setting Up Git & SSH Authentication on VPS

---

## Goal

Prepare the VPS so it can securely access my private GitHub repositories without entering my GitHub password every time.

---

## What I Did

- Installed Git
- Configured Git username and email
- Generated a new SSH key pair using `ed25519`
- Added the public key to my GitHub account
- Verified SSH authentication with GitHub

---

## Commands Learned

```bash
sudo apt install git -y

git --version

git config --global user.name "Aditya Kumar"

git config --global user.email "email@example.com"

ssh-keygen -t ed25519 -C "email@example.com"

cat ~/.ssh/id_ed25519.pub

ssh -T git@github.com
```

---

## New Things I Learned

### Git Configuration

Git stores my identity (name and email) globally so every commit is associated with me.

---

### SSH Key Pair

Generating an SSH key actually creates two keys:

- **Private Key (`id_ed25519`)** → Stays only on my VPS. Never share it.
- **Public Key (`id_ed25519.pub`)** → Safe to share. Added to GitHub.

GitHub uses these keys to verify that the VPS is authorized to access my repositories.

---

### Passwordless Authentication

Instead of entering my GitHub username and password every time, the VPS now authenticates automatically using the SSH key pair.

---

### Authentication Test

Running

```bash
ssh -T git@github.com
```

doesn't open a shell on GitHub.

Instead, it verifies whether the SSH key has been correctly registered. Receiving the message:

> The authenticity of host 'github.com (140.82.121.3)' can't be established.
> ED25519 key fingerprint is SHA256:+DiY3wvvV6TuJJhbpZisF/zLDA0zPMSvHdkr4UvCOqU.
> This key is not known by any other names.
> Are you sure you want to continue connecting (yes/no/[fingerprint])? yes
> Warning: Permanently added 'github.com' (ED25519) to the list of known hosts.
> Hi archaditya! You've successfully authenticated, but GitHub does not provide shell access.

confirmed that my VPS can securely communicate with GitHub.

---

## Why This Matters

Now I can clone, pull, and push repositories from the VPS securely without using passwords or personal access tokens.

---

## What's Next

- Clone the ByteVault repository on the VPS.
- Install Docker Compose.
- Deploy ByteVault using Docker Compose.
- Expose the application with Nginx and connect my domain.
