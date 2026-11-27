# Configure AI and publish an application

Use this guide after setting up Trustant on [Windows](setup-windows.md),
[macOS](setup-macos.md), or [Ubuntu](setup-ubuntu.md). Choose an AI provider
for the Trustant assistant, test your app on the development cluster, then
publish it to your production OpenServerless host. All three AI modes use
the same publishing procedure. If your app calls an AI service at runtime,
configure that service for the app in production as well.

## Private AI

1. Make an OpenAI-compatible endpoint reachable from the Trustant environment.
   Use the full base URL ending in `/v1`, for example
   `https://models.example.com/compatible-mode/v1`. Have the endpoint's API
   key ready if it requires one.
2. Choose **Private AI** on first run or through **Configuration → Change
   Provider**. Enter the base URL and key in **Private AI endpoint**, then
   click **Save**.
3. Confirm the endpoint's model list appears. Under **Pi Model → Default
   Model**, select a model that supports tool calls. Click **Save & Configure**
   and confirm the model test passes.
4. Follow [Publish the application](#publish-the-application). If the app uses
   the same endpoint, check that the production runtime can reach it and has
   its own production credentials where required.

## Cloud AI

1. Have an Ollama Cloud account with access to the cloud model you want to use.
   The **Cloud AI** card initially uses the Ollama server inside the Trustant
   environment.
2. Choose **Cloud AI**. If Trustant asks you to sign in, open the Ollama Cloud
   link, complete sign-in, then return and click **Retry**.
3. Confirm the model test passes. To select a different model, open
   **Configuration**, choose a tool-capable **Default Model** under **Pi Model**,
   then click **Save & Configure**. If the test reports exhausted capacity,
   check the signed-in account's access to that model.
4. Follow [Publish the application](#publish-the-application). For an app that
   calls Ollama Cloud itself, configure and test the app's own connection from
   the production runtime.

## Sovereign AI

1. Have access to Trustant registration and an available credit balance for
   Regolo.AI inference. The operator must enable `ENABLE_REGOLO=1` in the
   Trustant server environment and restart the server if the flag changes.
2. Choose **Sovereign AI** and complete the Trustant sign-up dialog.
3. Confirm the model test passes and the application list opens. To change
   the default model, use **Configuration → Pi Model**, then click
   **Save & Configure** and confirm the test succeeds with that model.
4. Follow [Publish the application](#publish-the-application). For an app that
   calls an inference API itself, configure and verify its production
   endpoint and credentials.

## Publish the application

1. Open the app with **Edit**, check the preview, then use **Commit** to save
   your changes. Use **Open Dev** to check the app on the development cluster.
2. Have the production OpenServerless API host, user, and password ready. With
   the default license enforcement, install a valid license under
   **Configuration** and check that it covers the destination's scheme, host,
   and port. The [license guide](../spec/14-license.md#local-host-exemption--host-gate-only)
   describes the host check for local destinations.
3. Select **Publish** from the application list. Enter **API Host**, **User**,
   and **Password** in **Publish to Production**. If a license host has been
   configured and you need an app user there, use **Create App on** followed by the host name in
   the same dialog. Supply the deployment's environment values and resolve any
   imports requested for that production host.
4. Click **Publish** and wait for **Published successfully**. Open the
   production URL from the target environment and test the app's main flow,
   including any services or model calls it makes. A successful assistant model
   test checks the Trustant workspace. The production URL check verifies the
   deployed app.

The procedure was checked against the Trustant UI, publishing code, and
specifications. Live tests with provider accounts and an end-to-end production
publish were outside this documentation check. See the [provider selection](../spec/1-index.md)
and [publishing](../spec/6-publish.md) specifications for implementation details.
