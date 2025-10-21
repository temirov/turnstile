# Front-end integration example

This walkthrough shows how a browser application can call the protected
`llm-proxy` service without ever handling the shared `SERVICE_SECRET`. The
ETS service issues a short-lived access token bound to the browser’s
DPoP key, and ETS forwards the request to `llm-proxy` while injecting
the secret server-side.

## 1. Import the ETS SDK

Include the browser SDK that ETS serves at `/sdk/tvm.mjs`. It generates a
P-256 keypair, mints short-lived access tokens, and attaches DPoP proofs
to every request.

```html
<script type="module">
  import { createGatewayClient } from "https://ets.mprlab.com/sdk/tvm.mjs";

  const gatewayClient = createGatewayClient({
    baseUrl: "https://ets.mprlab.com",
    apiPath: "/api" // ETS forwards to llm-proxy via backend config
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
    }
  });
</script>
```

The browser never sends the `SERVICE_SECRET`. ETS validates the origin,
mints a token for the browser’s DPoP key, injects the secret via the `key`
query parameter, and relays the request to `llm-proxy`.

## 2. How `ets.mprlab.com` is wired

The `tools/mprlab-gateway` repo’s `ets` branch defines the production wiring:

- `docker-compose.yml` runs `llm-ets` (the ETS container) alongside `llm-proxy`.
  The `llm-ets` service depends on `llm-proxy` and loads `.env.ets`, which
  includes:

  ```dotenv
  UPSTREAM_BASE_URL=http://llm-proxy:8080
  UPSTREAM_SERVICE_SECRET=replace-with-llm-proxy-service-secret
  ```

  This ensures every ETS call is forwarded to `llm-proxy` on the private Docker
  network with the correct shared secret.

- `Caddyfile` declares:

  ```caddy
  ets.mprlab.com {
      import common_site
      reverse_proxy llm-ets:8080 {
          transport http {
              dial_timeout            10s
              response_header_timeout 80s
              read_timeout            80s
          }
      }
  }
  ```

  With DNS pointing `ets.mprlab.com` at the Caddy host, HTTPS traffic is
  terminated at Caddy and forwarded to the ETS container. Because ETS itself
  targets `llm-proxy:8080`, the chain `browser → ets.mprlab.com → llm-ets →
  llm-proxy` works without exposing the upstream secret.

## 3. CORS and origins

Make sure ETS’s `ORIGIN_ALLOWLIST` includes every web origin that will call it
(for example, `https://loopaware.mprlab.com`). The example above uses
`fetchResponse` with `method: "GET"` so no request body is transmitted; the SDK
automatically manages token caching and DPoP proofs.
