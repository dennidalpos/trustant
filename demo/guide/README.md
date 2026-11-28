# Creation-mode guides for the Demo

Seven small apps in the Demo workbench each explain one way of creating an app in
Trustant, and show what happens underneath. Every app serves `/guida.html` with the
same structure: Intento, Cosa fai tu, Cosa succede sotto, Risultato, Verifica dal vivo.

| App | Mode explained | Built from |
|---|---|---|
| `guidaprovider` | provider choice | starter `trureact` |
| `guidastarter` | New application, own starter | own starter repository |
| `guidatruchat` | New application, catalog starter | starter `truchat` |
| `guidatemplate` | From a template | template `truingest` |
| `guidashared` | Share: Export and Import | own starter repository |
| `guidapublish` | Publish, Redeploy, Clean | own starter repository |
| `provachecklist` | working with the assistant | own starter repository |

## Files

- `guide-kit.css`, `guide-kit.js`: shared styling, navigation between the guides (the app
  name is the first label of the host) and the live probes. No dependencies.
- `build_guides.py <out-dir>`: writes `<out-dir>/<app>/guida.html` from one template and the
  per-app content. Messages quoted in "Cosa succede sotto" come from the server log; the rest
  follows `spec/`.
- `install.sh`: runs the generator and copies the pages and the kit into each workbench.

## Use

```bash
./install.sh                       # WORKBENCH=... to change the root
cd "$WORKBENCH/<app>" && ops ide login && ops ide deploy
```

Run `ops ide login` before each deploy: the login is global and decides the target user.
Do not print the deploy output unfiltered, action parameters can contain tokens.

The apps themselves are created from Trustant (Add App); a new app has no workbench until
Edit is pressed in the app list. The links to `/guida.html` in the React apps' headers are
small edits in the app sources and are not part of this kit.
