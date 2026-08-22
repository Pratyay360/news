---
id: d6uExHe1NUk
title: getting started
description: Getting started with Newsy
author: Pratyay360
visibility: "public"
created: 2026-08-22T17:45:42+00:00
---

This guide walks you through running Newsy and wiring it up to your repositories. 
There are two methods:
  - Run it locally for development
  - Deploy it to a serverless platform like Vercel for production

A video version of the local setup is available in the [greenfield setup guide](/video.html).

## Greenfield deployment

Clone the repository:

```bash
git clone git@github.com/pratyay360/newsy.git
```

We use [mise](https://mise.jdx.dev) to keep the development environment consistent and easy to manage:

```bash
mise deps
```

Build and run the Go binary:

```bash
go build
./newsy
```

Open [http://localhost:3000](http://localhost:3000). Copy that URL into the 
**Webhook URL** field when you create the GitHub App (give it a name and continue).

If you hit errors because of local host related issues kindly expose it with a 
tunneling service such as [ngrok](https://ngrok.com), [cf-tunnel](https://developers.cloudflare.com/cloudflare-one/networks/connectors/cloudflare-tunnel/), [zrok2](https://zrok.io) or simply use ssh with [localhost.run](http://localhost.run/) 
if not interested in downloading a separate cli tool.

For example:

```bash
ssh -R 80:localhost:3000 localhost.run
```

Open the url [tunnel url](https://xyz.localhost.run) in your browser.
it should look something like this: if not then watch my [video guide](/video.html)
or try with a different tunneling service.

![Webhook configuration](https://s6.imgcdn.dev/YgEOzv.png)

Create a app name(preferably something unique)

Put the tunnel URL (for example `https://xyz.localhost.run`) into the webhook urlfield.
If everything worked for you, a `.env` file will be created in your cwd. Keep it safe it contains your app's secrets.

The generated `.env` should look something like this:

```dotenv
APP_ID=""
WEBHOOK_SECRET=""
PRIVATE_KEY=""
```

**Important:** install the app on specific repositories only not on "all repositories".

Add these two variables yourself, and make sure the bot is installed on both repositories:

```dotenv
TRACK_REPO=""
DEST_REPO=""
```

```dotenv
ISSUE_TITLE = "" | optional default: "announcement"
ISSUE_LABEL = "" | optional default: "newsletter"
```

- `TRACK_REPO` — the repository Newsy watches for content changes.
- `DEST_REPO` — the repository where newsletter issues are created.

You can set both to the same repository (that's what I have done in the video).

![Repository selection](https://s6.imgcdn.dev/YgHyXO.png)

## Deploy on Vercel

[![Deploy with Vercel](https://vercel.com/button)](https://vercel.com/new/clone?repository-url=https%3A%2F%2Fgithub.com%2FPratyay360%2Fnewsy)

Once deployed, update your GitHub App's URLs from `*.localhost.run` to your
apps domain: `*.vercel.app`.

See the [Vercel setup guide](vercel-setup-guide.html) for the exact steps.

> Only use the deploy button if you already have a GitHub App and remember its credentials.
Otherwise, follow the [Green field](#greenfield-deployment) steps first to create one.

## Cost of running this

Vercel offers generous limits for serverless functions also there are many cloud
providers offering serverless functions as a service, also you can run the bot on
any cloud provider's serverless runtime — even on the ones that only support 
JS/TS functions (shh, by compiling the Go code to Wasm with [TinyGo](https://tinygo.org)).

Because it's a compiled Go binary, there is no runtime dependencies once built, 
and it's memory-efficient, as it's compiled machine code. You can comfortably stay
within invocation memory and duration limits.

## Try my newsletter

Subscribe to my own newsletter to see it in action:

<a href="https://github.com/Pratyay360/pratyay/issues/13" target="_blank" rel="noopener noreferrer" style="display:inline-block;font-family:-apple-system,BlinkMacSystemFont,&#39;Segoe UI&#39;,sans-serif;font-weight:600;line-height:1.25;text-align:center;text-decoration:none;cursor:pointer;padding:7px 16px;font-size:14px;border-radius:6px;background:#0969da;color:#ffffff;border:1px solid rgba(27, 31, 36, 0.15);box-shadow:none;transition:background .15s ease, transform .15s ease" >Subscribe on GitHub</a>
## Generate your own button

Create a subscription button for your account at [https://newsy.surge.sh/](https://newsy.surge.sh/).

You can style and customize the button using [Primer CSS](https://cdnjs.com/libraries/Primer).

> **Security tip:** after the bot publishes its first newsletter issue, lock the
> conversation so no one else can post in it and exploit your newsletter.

![Locking the conversation](https://s6.imgcdn.dev/YgXhaN.png)

Refer to the [required permissions](list-of-all-permission.html) page while
setting things up.
