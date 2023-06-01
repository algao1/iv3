<div align="center">

# iv3
Yet another T1D management solution.

**Third time's the charm. Right?**

<img src="./.media/ghost_gopher.png" width="250" height="250">

*This image was generated using Midjourney.*

</div>

## Quickstart
To get started, run:
```
task build
task all
```

To perform a backup or restore, run:
```
task idb-backup
task idb-restore
```

## Overview
This project is primarily meant for personal use, the steps required to set up **iv3** are not as accessible as those for [Nightscout](https://nightscout.github.io/).

Primarily meant to be used in conjunction with a Retool dashboard, as shown below.

<a href=".media/iv3_desktop_retool.png"><img src=".media/iv3_desktop_retool.png" height="250"/></a>
<a href=".media/iv3_mobile_retool.png"><img src=".media/iv3_mobile_retool.png" height="250"/></a>

## What's Different?
This time around, I want to:
- Avoid having to write/support a Python graph rendering service
- Avoid having to maintain Discord functionality
- Redesign some architecture, hopefully making it easier to maintain

So, I have decided to keep it more simple, and rely more heavily on third-party integrations for certain functionality.
- Use Retool for realtime dashboards and input
- Use InfluxDB instead of MongoDB for storing timeseries data
- More robust and periodic backups to blob storage
- Better CI/CD, fearless and seamless deployments
- Make the process easier to spin-up experiments for data pipelines

For previous versions, see [ichor](https://github.com/algao1/ichor) and [iv2](https://github.com/algao1/iv2).

## Roadmap
- Better analysis on day-to-day patterns
- Sustained high glucose alerts
- Better low glucose detection
	- Factors in previous carbs and insulin values
- And more to come!

## Dependencies
- [Task](https://taskfile.dev/)
- [Retool](https://retool.com/)
- [ntfy](https://ntfy.sh/)
- Docker

## Setup & Config
Setup will also require a few other things, mostly left as a note to myself. See `example-config.yaml` for more details.
- A .env file for things that have to be used by Task or docker compose
    - `INFLUXDB_TOKEN=...` to access InfluxDB
    - `IV3_ENV=dev` for dev environment
- A config.yaml file for application-level settings
    - Dexcom, Spaces keys and secrets
    - Insulin types
	- Alerting configs
- A domain name, and an SSL certificate for HTTPS
    - This will be needed for authentication, and for Retool API integrations
    - `certfile.crt`, `keyfile.key`