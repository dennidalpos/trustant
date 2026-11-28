# Prompts to recreate the Demo test apps

These prompts rebuild the guide apps listed in [README.md](README.md) from the
assistant of a new Trustant app. They do not change how Trustant sets apps up:
each app is created from Add App with the mode named below, and the assistant
only edits the app's own sources. Nothing here is copied from the upstream
repository; the guides are original text.

Create the app first (name in the table), press Edit once so the workbench
exists, then paste the common prompt followed by the app-specific one.

## Common prompt

```text
Build a page /guida.html for this app, in Italian, with five numbered sections:
1. Intento, 2. Cosa fai tu, 3. Cosa succede sotto, 4. Risultato, 5. Verifica dal vivo.
Use plain HTML and CSS with no dependencies; keep it accessible (landmarks, one h1,
labelled controls, visible focus). In section 5 show a table the browser fills in at
load time by reading this app's own published URL and the host it runs on, never typed
by hand. Link to the other guide apps by replacing the first label of the current host
with their name. Do not touch deploy configuration, secrets, or action parameters, and
do not print deploy output unfiltered.
```

## Per app

| App | Created with | Prompt |
|---|---|---|
| `guidaprovider` | New application, starter `trureact` | Explain how to choose the AI provider (Private, Cloud, Sovereign) when creating an app, what each choice stores, and where the React entry point, Vite output and OpenServerless actions live. Add a second page `/trureact.html` on creating an app from the TruReact starter. |
| `guidastarter` | New application, My Application Starter | Explain starting from your own GitHub repository: name rules (6–20 alphanumeric, starts with a letter), `org/repo` format, the bare working repository, the first launch on Edit. Add a live check that validates a typed name and repository and shows the derived paths and host. Keep the starter's own technical page as `/riferimento.html`. |
| `guidatruchat` | New application, starter `truchat` | Explain starting from a catalog starter: what the starter provides, which AI values it needs, and what the first launch does. |
| `guidatemplate` | From a template, `truingest` | Explain creating an app from a template: what is copied, which parameters the actions take, and how redeploy relates to them. Never place a credential in an action parameter. |
| `guidashared` | New application, My Application Starter | Explain Export and Import of shared variables between apps and what is and is not carried over. |
| `guidapublish` | New application, My Application Starter | Explain Publish, Redeploy and Clean: what each does to the cluster and to the repository, and what is irreversible. |
| `provachecklist` | New application, My Application Starter | Build a 34-item checklist for working with the assistant (filters, persistence in `localStorage`, keyboard use), with each item checkable and the state saved across reloads. |

## Notes

- Run `ops ide login` before each `ops ide deploy`: the login is global and decides the target user.
- The prompts are a starting point; the assistant output differs per run and per model.
- Keep the apps in the Demo workbench. They are tests, not part of the product.
