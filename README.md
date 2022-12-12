# Pønskelisten
[![Github Stars](https://img.shields.io/github/stars/aunefyren/poenskelisten?style=for-the-badge)](https://github.com/aunefyren/poenskelisten)
[![Github Forks](https://img.shields.io/github/forks/aunefyren/poenskelisten?style=for-the-badge)](https://github.com/aunefyren/poenskelisten)
[![Docker Pulls](https://img.shields.io/docker/pulls/aunefyren/poenskelisten?style=for-the-badge)](https://hub.docker.com/r/aunefyren/poenskelisten)
[![Newest Release](https://img.shields.io/github/v/release/aunefyren/poenskelisten?style=for-the-badge)](https://github.com/aunefyren/poenskelisten/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/aunefyren/poenskelisten?style=for-the-badge)](https://go.dev/dl/)

<br>
<br>

## Introduction - What is this? 🎁

A website-based application for sharing and collaborating on wishlists and presents. The main goal is to allow the sharing of wishlists and the claiming gift ideas without the recipient knowing what they are receiving.

<br>
<br>

![Image showing the wishlist section of Pønskelisten.](https://raw.githubusercontent.com/aunefyren/poenskelisten/main/web/assets/images/wishlists_example.png?raw=true)

<br>
<br>

Notable features:
- Group people using, well, groups
- Create wishlists, with wishes
- Have multiple wishlists shared with multiple groups. This allows you to have synchronized wishlists for multiple people, without them having to see each other's wishlists
- Wish claiming. Someone can claim a gift on a wishlist they are allowed to see. Anyone else who can see that wishlist then knows that gift idea is taken. The owner can't see this of course

Known issues:
- There currently is no admin interface to add invitation codes
- No email account confirmation through SMTP
- Can't leave groups someone else added you too
- Can't add additional wishlist owners
- Can't edit group/wishlist/wish details
- Can't change account information
- UI Can be a bit cluttered on smaller screens such as phones
- Wishlist expiration is not utilized yet

<br>
<br>

![Image showing the wishlist section of Pønskelisten.](https://raw.githubusercontent.com/aunefyren/poenskelisten/main/web/assets/images/claim_example.png?raw=true)

<br>
<br>

## Installation - A bit cumbersome 😰

I recommend using Docker honestly.
<br>
<br>

### 1. You need a database

A MySQL database specifically. In the future, this process will be streamlined and the different databases supported by the DB module will be added. But for now, setup a MySQL database that Pønskelisten can reach and log into.

If you are hosting this manually you could download XAMPP and just click "start" on the DB feature. If you are using Docker, use the MySQL Docker image. There is a Docker compose example further down.

Create a table for Pønskelisten (Docker image does this automatically), and use that name in the next step.

<br>
<br>

### 2. Configure the ```/files/config.json``` file

Edit the config file so it can reach the MySQL database. There is no admin interface currently so this must be done manually in the file. The timezone is also necessary, but the private key should populate automatically. The SMTP settings do nothing currently.

<br>
<br>

### 3. Start Pønskelisten

Either compile your chosen branch/tag with Go:

```
$ go build
$ ./poenskelisten
```
... or download a pre-compiled release and start the application.

It should say whether or not it started and managed to connect to the database. Pønskelisten is now up and running.

<br>
<br>

#### Startup flags

You can use startup flags to generate values to populate the configuration file with. They are only used if the configuration file doesn't have a value to prioritize. The moment the configuration file has values, these flags are useless. 

The exception is ```generateinvite```, which will generate a new, random invitation code at each usage.

<br>
<br>

| Flag | Type | Explaination |
|-|:-:|-:|
| port | integer | Which port Pønskelisten starts on |
| timezone | string | The timezone Pønskelisten uses. Given in the TZ database name format. List can be found [here](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) |
| generateinvite | string (true/false) | If Pønskelisten should generate an invitation code on startup |
| dbip | string | The connection address Pønskelisten uses to reach the database |
| dbport | integer | The port Pønskelisten can reach the database at |
| dbname | string | The name of the table within the database |
| dbusername | string | The username used to autnenicate with the database |
| dbpassword | string | The password used to autnenicate with the database |

<br>
<br>

To use a flag, just start the compiled Go program with additonal values. Such as:

```
$ ./poenskelisten -p 7679
```

<br>
<br>

### 4. Be able to alter the DB

Once again, there is no admin interface. To sign up for the website you need an invitation code. If you used the ```generateinvite``` flag you can find an invitation code in the log/console. 

 If not, you need to alter the database table to add the invitation code. Cumbersome, I know. 

<br>
<br>

I recommend installing PHPMyAdmin (DB interface) either as a Docker image or locally (it comes pre-packaged in XAMPP).

After accessing the DB through an interface, or by just running SQL commands, add an invitation code to the table ```invitations```. You should now be able to sign up using the code at Pønskelisten frontend.

By default you can find the frontend at ```localhost:8080```.

<br>
<br>

You need an invitation code for every user who wants to sign up.

<br>
<br>

MySQL example command for adding an invitation code called "RANDOMCODE":

```
INSERT INTO `invites` (`id`, `created_at`, `updated_at`, `deleted_at`, `invite_code`, `invite_used`, `invite_recipient`, `invite_enabled`) VALUES (NULL, CURRENT_TIME(), CURRENT_TIME(), NULL, 'RANDOMCODE', '0', NULL, '1'); 
```

Be prepared to access the DB every time a user manages to screw up their e-mail while signing up or someone needs an invitation code.

<br>
<br>

## Docker

### Environment variables

All the startup flags in the table given previously can be used as environment variables. Do keep in mind that the flags, and in turn the environment variables, are only used if the value is not already defined in the configuration file. 

The only exception is the ```generateinvite```, but consider removing this environment variable after usage so you don't generate a new code at every restart.

<br>
<br>

### Docker-Compose example
It has Pønskelisten, MySQL DB and PHPMyAdmin.
```
version: '3.3'
services:
  db:
    image: mysql:5.7
    container_name: poenskelisten-db
    restart: unless-stopped
    environment:
      # The table name you chose
      MYSQL_DATABASE: 'ponske'
      # User, so you don't have to use root 
      MYSQL_USER: 'myuser'
      # Please switch this password
      MYSQL_PASSWORD: 'mystrongpassword' 
      # Password for root access, change this too
      MYSQL_ROOT_PASSWORD: 'root' 
    networks:
      - db
    expose:
      - '3306'
    # Where our data will be persisted
    volumes:
      - ./db/:/var/lib/mysql/ # Location of DB data
  poenskelisten:
    container_name: poenskelisten-app
    image: aunefyren/poenskelisten:latest
    restart: unless-stopped
    networks:
      - db
    depends_on:
      - db
    volumes:
      - ./data/:/app/files/
    ports:
      - '8080:8080'
    environment:
      # Generate an unused invite code on startup
      # Remove this value to avoid continuous code-generation
      generateinvite: true
      # The container will only respect these ENV if they are empty in the config.json
      # Useful for first setup
      port: 8080
      timezone: Europe/Oslo
      dbip: db
      dbport: 3306
      dbname: ponske
      dbusername: myuser
      dbpassword: mystrongpassword
  phpmyadmin:
    image: phpmyadmin:latest
    restart: unless-stopped
    environment:
      - PMA_ARBITRARY=1
      # DB table
      - PMA_HOST:ponske 
      # Root password
      - MYSQL_ROOT_PASSWORD:root 
      # Timezone
      - TZ=Europe/Oslo 
    container_name: poenskelisten-phpmyadmin
    ports:
      - 80:80
    depends_on:
      - db
    networks:
      - db

networks:
  db:
    external: false
```

<br>
<br>

## FAQ - What u mean??? 😕

<b>What does Pønskelisten mean?</b><br>
Just a clever Norwegian wordplay that doesn't translate to English at all. A wishlist is called a 'ønskeliste' in Norwegian, and the verb to 'pønske' means to plot and plan. Therefore, Pønskelisten.

<br>
<br>

<b>Can you please remove the need to manage the DB directly?</b><br>
Yeah yeah, it's coming.

<br>
<br>

## The End - Does it work? 🪛
Well, this is in early development and making it user-friendly and worth hosting is hard and takes time. It is certainly functional, but it isn't necessarily what you need or want. Feel free to add feedback or feature requests in the form of GitHub Issues.
