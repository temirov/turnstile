# Front-end integration example

This walkthrough shows how a browser application can call the protected
`llm-proxy` service without ever handling the shared `SERVICE_SECRET`. The
Turnstile service issues a short-lived access token bound to the browser’s
DPoP key, and Turnstile forwards the request to `llm-proxy` while injecting
the secret server-side.

## 1. Load the Turnstile widget

Include the Turnstile widget on the page and render it when the user is ready
to submit a prompt:

```html
<script src="https://turnstile.mprlab.com/widget.js" async defer></script>

<div id="turnstile-placeholder"></div>

<script>
  window.turnstileWidgetId = null;
  window.onload = () => {
    window.turnstileWidgetId = turnstile.render("#turnstile-placeholder", {
      sitekey: "1x0000000000000000000000000000000AA"
    });
  };
</script>
```

## 2. Create the Turnstile client

Import the SDK that Turnstile serves at `/sdk/tvm.mjs`. Configure the base
URL and provide a function that returns the current Turnstile response.

```html
<script type="module">
  import { createGatewayClient } from "https://turnstile.mprlab.com/sdk/tvm.mjs";

  const gatewayClient = createGatewayClient({
    baseUrl: "https://turnstile.mprlab.com",
    apiPath: "/api", // Turnstile forwards to llm-proxy / via backend config
    turnstileTokenProvider: () => window.turnstile.getResponse(window.turnstileWidgetId)
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
      throw new Error("Turnstile error " + response.status + ": " + (await response.text()));
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
      window.turnstile.reset(window.turnstileWidgetId);
    }
  });
</script>
```

The browser never sends the `SERVICE_SECRET`. Turnstile verifies the widget
response and DPoP, injects the secret via the `key` query parameter, and relays the
request to `llm-proxy`.

## 3. CORS and origins

Make sure Turnstile’s `ORIGIN_ALLOWLIST` includes every web origin that will
call it (for example, `https://loopaware.mprlab.com`). The example above uses
`fetchResponse` with `method: "GET"` so no request body is transmitted; the
SDK automatically manages token caching and DPoP proofs.
