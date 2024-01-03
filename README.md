<div align="center">

# iv3
Yet another type 1 diabetes management solution.

**Third time's the charm. Right?**

<img src="./.media/ghost_gopher.png" width="250" height="250">

*This image was generated using Midjourney.*

</div>

## ToC
- [Overview](#overview)
- [Quickstart](#quickstart)
- [Development and Deployment](#development-and-deployment)
- [What's Different](#whats-different)
- [Roadmap](#roadmap-todo)

## Overview
This project is primarily meant to be used in conjunction with a Retool dashboard, as shown below.

<div style="text-align: center;">
	<a href=".media/iv3_desktop_retool.png"><img src=".media/iv3_desktop_retool.png" height="250"/></a>
	<a href=".media/iv3_mobile_retool.png"><img src=".media/iv3_mobile_retool.png" height="250"/></a>
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

To develop locally, run:
```
task go
```

See `Taskfile.yaml` for more specifics.

## Development and Deployment
This project is primarily meant for personal use, so the steps outlined here are left mostly as a note to myself. The structure and layout is very opinionated and tailored to my usecases.

See `example-config.yaml` for more specific details.

Setup will also require a few other things:
- A `.env` file for things that are used by Task or docker compose
    - `INFLUXDB_TOKEN=...` to access InfluxDB
    - `IV3_ENV=dev` for dev environment
- A `config.yaml` file for application-level settings
    - Dexcom, DigitalOcean Spaces keys and secrets
    - Insulin and alerting configs
- A domain name, and SSL certificate for HTTPS (needed for Retool)
    - This will be needed for authentication, and for Retool API integrations
    - `certfile.crt`, `keyfile.key`
	- **Note:** At this moment, these files need to be located inside `_iv3_ssl` since it is mounted onto iv3, and will not work otherwise

### InfluxDB
The configuration and data for InfluxDB are mounted on `_iv3_config` and `_iv3_data` respectively, remember to create those! Additionally, when starting the InfluxDB instance for the first time, we need to register and create an API token, see [**here**](https://hub.docker.com/_/influxdb) for details.

Once that is done, remember to add it to the `config.yaml` file.

### ntfy

TBD.

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

## Roadmap (Todo):
- Different predictors for low glucose
- Target glucose for different times of the day
- Factor in insulin to carbs ratio
- ChatGPT integration
- Automatic S3 bucket cleanup (retention)
- Update Retool graphs and dashboard

## Dependencies
- [Task](https://taskfile.dev/)
- [InfluxDB](https://www.influxdata.com/)
- [Retool](https://retool.com/)
- [ntfy](https://ntfy.sh/)
