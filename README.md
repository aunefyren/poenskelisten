# Pønskelisten

[![Github Stars](https://img.shields.io/github/stars/aunefyren/poenskelisten?style=for-the-badge)](https://github.com/aunefyren/poenskelisten)
[![Github Forks](https://img.shields.io/github/forks/aunefyren/poenskelisten?style=for-the-badge)](https://github.com/aunefyren/poenskelisten)
[![Docker Pulls](https://img.shields.io/docker/pulls/aunefyren/poenskelisten?style=for-the-badge)](https://hub.docker.com/r/aunefyren/poenskelisten)
[![Newest Release](https://img.shields.io/github/v/release/aunefyren/poenskelisten?style=for-the-badge)](https://github.com/aunefyren/poenskelisten/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/aunefyren/poenskelisten?style=for-the-badge)](https://go.dev/dl/)

<br>

[![Donate](https://img.shields.io/badge/PayPal-Buy%20me%20coffee-blue?style=for-the-badge)](https://www.paypal.com/donate/?hosted_button_id=YRKMNM4S8VNBS)

Like the project? Have too much money? Buy me a coffee or something! ☕️

---

## What is Pønskelisten? 🎁

A self-hosted web app for creating, sharing and collaborating on wishlists — *without ruining the surprise*.  
Share gift ideas, see which ones are already taken, and avoid the awkward “oh… you also bought that…” moment.

### Main Features
- Create wishlists and add wishes
- Collaborate with friends & family
- Create groups to share wishlists with multiple people
- Claim wishes anonymously (others see it's taken — owner does not)
- First registered user becomes admin automatically

### Known Limitations
- UI is not yet optimized for small screens

<br>

![Wishlists screenshot](https://raw.githubusercontent.com/aunefyren/poenskelisten/main/.github/assets/wishlists.jpg?raw=true)

<br>

---

## 🚀 Installation

Pønskelisten is flexible to host. Choose your path:

### **Step 1: Choose how to run it**

| Method | Difficulty | Recommended for |
|--------|-------------|------------------|
| **Docker** | ⭐ Easiest | Most users |
| Download executable | Easy | Desktop/server users |
| Build from source | Medium | Developers |

### **Step 2: Choose your Database**

Pønskelisten currently supports:

| Database | Status |
|----------|--------|
| PostgreSQL | ✅ Fully supported |
| MySQL | ✅ Fully supported |

> **No direct DB management required anymore 🎉**  
Pønskelisten handles setup on first run.

---

## 🧩 Configuration (Recommended: Environment Variables)

You can configure Pønskelisten in **three different ways**:

| Method | Ideal for | Notes |
|--------|------------|--------|
| **Environment variables** ✅ | Docker, production | Recommended |
| Startup flags | Local runs, executables | Overrides config.json |
| config.json | Manual configs | Pønskelisten generates one at first run |

All configuration keys are identical across methods.

---

### 📍 Available Configuration Options

| Key | Type | Description |
|-----|-------|--------------|
| port | int | Port to run on (default: `8080`) |
| externalurl | string | Public URL of the instance |
| timezone | string | E.g. `Europe/Oslo` |
| environment | string | `prod` or `test` |
| name | string | Display name of the app |
| generateinvite | bool | Generate an invite code on startup |
| dbtype | string | `postgres` or `mysql` |
| dbip | string | DB host |
| dbport | int | DB port |
| dbusername | string | DB username |
| dbpassword | string | DB password |
| dbname | string | Database name |
| dbssl | bool | Use SSL for DB |
| disablesmtp | bool | Disable email verification |
| smtphost | string | SMTP host |
| smtpport | int | SMTP port |
| smtpusername | string | SMTP user |
| smtppassword | string | SMTP password |
| smtpfrom | string | Sender email address |

---

## 🐳 Docker Setup

### **Minimal docker-compose.yml (recommended)**

```yaml
version: "3.3"
services:
  db:
    image: postgres:16
    restart: unless-stopped
    environment:
      POSTGRES_DB: poenskelisten
      POSTGRES_USER: myuser
      POSTGRES_PASSWORD: mypassword
    volumes:
      - ./db:/var/lib/postgresql/data

  poenskelisten:
    image: aunefyren/poenskelisten:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      dbtype: postgres
      dbip: db
      dbport: 5432
      dbname: poenskelisten
      dbusername: myuser
      dbpassword: mypassword
      timezone: Europe/Oslo
      generateinvite: true
    depends_on:
      - db
```
Remove generateinvite after first run to keep the same code.

<br>
Optional Add-ons

Reverse proxy (Caddy, Traefik, Nginx)

Adminer or phpMyAdmin (if you like UI DB tools, but not required anymore)

## 🔑 Admin Access

First registered user becomes admin

Additional invite codes can be created in the admin panel

If you lose access: restart with generateinvite=true

## 🔧 Building from Source

Requires Go installed:
```bash
go build
./poenskelisten
```

Add flags to configure:
```bash
./poenskelisten -port 9000 -dbtype postgres -generateinvite true
```

## ❓ FAQ

### What does “Pønskelisten” mean?
A Norwegian wordplay. “Ønskeliste” = wishlist, “pønske” = plot/plan.
So… “The plotting list”.

### Is a demo available?
Not at the moment.

### Is mobile UI coming?
Yes — improvements planned.

## 🙌 Contributing

Contributions, issues and feature requests are welcome.
Feel free to open a Pull Request or Issue.

## ☕️ Donate

If you enjoy using Pønskelisten and want to support development:<br>
[Buy me a coffee](https://www.paypal.com/donate/?hosted_button_id=YRKMNM4S8VNBS)