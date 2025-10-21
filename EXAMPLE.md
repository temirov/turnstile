# Front-end integration example

This walkthrough shows how a browser application can call the protected
`llm-proxy` service without ever handling the shared `SERVICE_SECRET`. The
ETS service issues a short-lived access token bound to the browser’s
DPoP key, and ETS forwards the request to `llm-proxy` while injecting
the secret server-side.

## 1. Load the ETS widget

Include the ETS widget on the page and render it when the user is ready
to submit a prompt:

```html
<script src="https://ets.mprlab.com/widget.js" async defer></script>

<div id="ets-placeholder"></div>

<script>
  window.etsWidgetId = null;
  window.onload = () => {
    window.etsWidgetId = ets.render("#ets-placeholder", {
      sitekey: "1x0000000000000000000000000000000AA"
    });
  };
</script>
```

## 2. Create the ETS client

Import the SDK that ETS serves at `/sdk/tvm.mjs`. Configure the base
URL and provide a function that returns the current ETS response.

```html
<script type="module">
  import { createGatewayClient } from "https://ets.mprlab.com/sdk/tvm.mjs";

  const gatewayClient = createGatewayClient({
    baseUrl: "https://ets.mprlab.com",
    apiPath: "/api", // ETS forwards to llm-proxy / via backend config
    etsTokenProvider: () => window.ets.getResponse(window.etsWidgetId)
  });

  async function runPrompt(promptText, model = "gpt-4o") {
    const params = new URLSearchParams({
      prompt: promptText,
      model: model,
      web_search: "1"
    });

    const response = await gatewayClient.fetchResponse(null, {
      method: "GET",
      path: "/?" + params.toString()
    });
    if (!response.ok) {
      throw new Error("ETS error " + response.status + ": " + (await response.text()));
    }
    return await response.text();
  }

  document.querySelector("#submit-button").addEventListener("click", async () => {
    const prompt = document.querySelector("#prompt-input").value;
    try {
      const reply = await runPrompt(prompt);
      document.querySelector("#response").textContent = reply;
    } catch (error) {
      console.error("Failed to call llm-proxy", error);
    } finally {
      window.ets.reset(window.etsWidgetId);
    }
  });
</script>
```

The browser never sends the `SERVICE_SECRET`. ETS verifies the widget
response and DPoP, injects the secret via the `key` query parameter, and relays the
request to `llm-proxy`.

## 3. CORS and origins

Make sure ETS’s `ORIGIN_ALLOWLIST` includes every web origin that will
call it (for example, `https://loopaware.mprlab.com`). The example above uses
`fetchResponse` with `method: "GET"` so no request body is transmitted; the
SDK automatically manages token caching and DPoP proofs.
