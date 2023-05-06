<div align="center">

# iv3
Yet another integrated T1D management solution.

**Third time's the charm. Right?**

<img src="./.media/ghost_gopher.png" width="400" height="400">

*This image is generated using Midjourney.*

</div>

## Quickstart

## What's Different?
This time around, I want to:
- Avoid having to write/support a Python graph rendering service
- Avoid having to maintain Discord functionality (it is a nice-to-have)
- Redesign some architecture, to make it easier to maintain

So, I have decided to keep it more simple, and rely more heavily on third-party integrations for certain functionality.
- Use Retool for realtime dashboards and CRUD (insulin, etc.)
    - Use Discord only for notifications, keeps it low-code
- Use InfluxDB instead of MongoDB for storing timeseries data
- Better CI/CD, fearless and seamless deployments
- More robust backups, and backup to blob storage
- Make the process easier to spin-up experiments for data pipelines

## Architecture

## Dependencies
- Task
- Docker
- Retool

## Other
Setup will also require a few other things, mostly left as a note to myself.
- A .env file with the `INFLUXDB_TOKEN`
    - Generally used for things that have to be used by Task or docker compose
- A config.yaml file with certain configs
    - Generally only used for application-level settings
- A domain name, and an SSL certificate for HTTPS
    - This will be needed for authentication, and for Retool API integrations
    - `certfile.crt`, `keyfile.key`