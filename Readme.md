# What is this?

Simple bot for monitoring turns in
[terraforming mars](https://github.com/terraforming-mars/terraforming-mars)

# Building

```bash
go build
```

# How to use it?

Build from source. Edit config file.

Run it:

```bash
./marsbot -config config.yaml
```

After that, add the bot to a group chat as administrator or write it directly.
To start monitoring a new game, use command:

```
/watch@mars_bot https://rebalanced-mars.herokuapp.com/spectator?id=se4369075767e
```

Use spectator link (not a player link)

# Configuration

```yaml
token: <token> # telegram bot token
database: ./db.sqlite # database file
allowed_domains: # list of allowed mars game domains
  - rebalanced-mars.herokuapp.com
```
