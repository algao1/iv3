<div align="center">

# iv3
Yet another integrated T1D management solution.

**Third time's the charm. Right?**

<img src="./.media/ghost_gopher.png" width="400" height="400">

*This image is generated using Midjourney.*

</div>

## What's Different?
This time around, I want to:
- Avoid having to write/support a Python graph rendering service
- Avoid having to maintain Discord functionality (it is a nice-to-have)
- Redesign some architecture, to make it easier to maintain

So, I have decided to keep it more simple, and rely more heavily on third-party integrations for certain functionality.
- Use Retool for realtime dashboards and CRUD (insulin, etc.)
    - Use Discord only for notifications, keeps it low-code
- Use InfluxDB instead of MongoDB for storing timeseries data
- Better plan out how configurations are propogated/loaded in
- Better CI/CD, fearless and seamless deployments
- More robust backups, and actually backup to blob storage
- Make the process easier to spin-up experiments for data pipelines

## Architecture

## Dependencies
- Task
- Docker
- Retool

## Quickstart