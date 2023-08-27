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

To start monitoring a new game, use command:

```
/subscribe https://rebalanced-mars.herokuapp.com/player?id=pe4369075767e
```

To list subscriptions, use command:

```
/subscriptions
```

To unsubscribe, use command:

```
/unsubscribe pe4369075767e
```

# Configuration

```yaml
token: <token> # telegram bot token
database: ./db.sqlite # database file
allowed_domains: # list of allowed mars game domains
  - rebalanced-mars.herokuapp.com
```
