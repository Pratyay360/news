---
id: o-EQ8FDLxzg
title: Vercel Setup Guide
description: How to point your GitHub App at a Vercel deployment.
author: Pratyay360
created: 2026-08-22T08:42:09+00:00
visibility: "public"
---

If you followed the [Getting Started](getstarted.html) guide, your GitHub App's
webhook URL currently points at a local tunnel, something like [https://xyz.localhost.run](https://xyz.localhost.run)

Once you've deployed Newsy to Vercel, replace that tunnel URL with your Vercel
deployment URL, for example [https://xyz.vercel.app](https://xyz.vercel.app).

Set the webhook URL to:

```url
https://xyz.vercel.app/api/github/webhooks
```

[<img title="" src="https://vercel.com/button" alt="Deploy with Vercel" width="186">](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2FPratyay360%2Fnewsy)


![GitHub App webhook settings.png](https://s6.imgcdn.dev/YgJqs8.md.png)
