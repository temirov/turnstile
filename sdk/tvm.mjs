// tvm.mjs — Browser SDK for the Ephemeral Token Service (ets.mprlab.com) with DPoP + JWT gateway

/**
 * createGatewayClient(options)
 * options: {
 *   baseUrl: string,                       // e.g., "https://ets.mprlab.com"
 *   tokenPath?: string,                    // default "/tvm/issue"
 *   apiPath?: string,                      // default "/api"
 *   etsTokenProvider?: () => Promise<string> | string
 * }
 *
 * Returns: {
 *   postJson(payload: any, init?: {signal?: AbortSignal, path?: string}): Promise<any>
 *   fetchResponse(payload: any, init?: {signal?: AbortSignal, path?: string}): Promise<Response>
 * }
 */

export function createGatewayClient(options) {
  const normalizedOptions = normalizeOptions(options);
  const keyState = { cryptoKeyPair: null };
  const tokenState = { accessToken: null, expiresAtEpochSeconds: 0 };

  async function postJson(requestPayload, init) {
    const response = await fetchResponse(requestPayload, init);
    const contentType = response.headers.get("Content-Type") || "";
    if (!contentType.includes("application/json")) {
      const textBody = await response.text();
      throw new Error("Unexpected content type: " + contentType + " body=" + textBody);
    }
    const jsonBody = await response.json();
    if (!response.ok) {
      throw new Error("Gateway error " + response.status + ": " + JSON.stringify(jsonBody));
    }
    return jsonBody;
  }

  async function fetchResponse(requestPayload, init) {
    const effectivePath = init?.path || normalizedOptions.apiPath;
    const requestUrl = joinUrl(normalizedOptions.baseUrl, effectivePath);
    const methodName = (init?.method || "POST").toUpperCase();

    const cryptoKeyPair = await ensureKeyPair(keyState);
    const { accessToken } = await ensureAccessToken({
      normalizedOptions,
      cryptoKeyPair,
      tokenState
    });

    const dpopJwt = await createDpopJwt({
      requestUrl: requestUrl,
      httpMethod: methodName,
      cryptoKeyPair: cryptoKeyPair
    });

    const headers = {
      "Authorization": "Bearer " + accessToken,
      "DPoP": dpopJwt
    };
    const shouldSendBody = methodName !== "GET" && methodName !== "HEAD";
    if (shouldSendBody) {
      headers["Content-Type"] = "application/json";
    }

    if (init?.headers) {
      for (const [headerName, headerValue] of Object.entries(init.headers)) {
        headers[headerName] = headerValue;
      }
    }

    const response = await fetch(requestUrl, {
      method: methodName,
      headers: headers,
      body: shouldSendBody ? JSON.stringify(requestPayload || {}) : undefined,
      signal: init?.signal
    });
    return response;
  }

  return { postJson, fetchResponse };
}

/* ---------- internals ---------- */

function normalizeOptions(options) {
  if (!options || typeof options.baseUrl !== "string" || options.baseUrl.length < 4) {
    throw new Error("createGatewayClient requires { baseUrl }");
  }
  return {
    baseUrl: options.baseUrl.replace(/\/+$/, ""),
    tokenPath: options.tokenPath || "/tvm/issue",
    apiPath: options.apiPath || "/api",
    etsTokenProvider: options.etsTokenProvider
  };
}

async function ensureKeyPair(state) {
  if (state.cryptoKeyPair) return state.cryptoKeyPair;
  const generatedKeyPair = await crypto.subtle.generateKey(
    { name: "ECDSA", namedCurve: "P-256" },
    true,
    ["sign", "verify"]
  );
  state.cryptoKeyPair = generatedKeyPair;
  return generatedKeyPair;
}

async function ensureAccessToken({ normalizedOptions, cryptoKeyPair, tokenState }) {
  const marginSeconds = 20;
  const nowSeconds = Math.floor(Date.now() / 1000);
  if (tokenState.accessToken && tokenState.expiresAtEpochSeconds - nowSeconds > marginSeconds) {
    return { accessToken: tokenState.accessToken, expiresIn: tokenState.expiresAtEpochSeconds - nowSeconds };
  }

  const publicJwk = await crypto.subtle.exportKey("jwk", cryptoKeyPair.publicKey);
  const requestBody = {
    dpopPublicJwk: { kty: publicJwk.kty, crv: publicJwk.crv, x: publicJwk.x, y: publicJwk.y },
    etsToken: await resolveEtsToken(normalizedOptions.etsTokenProvider)
  };

  const tokenResponse = await fetch(joinUrl(normalizedOptions.baseUrl, normalizedOptions.tokenPath), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(requestBody)
  });
  if (!tokenResponse.ok) {
    const textBody = await tokenResponse.text();
    throw new Error("Token vending failed: " + tokenResponse.status + " " + textBody);
  }
  const tokenJson = await tokenResponse.json();
  const expiresInSeconds = Number(tokenJson.expiresIn || 0);
  tokenState.accessToken = tokenJson.accessToken;
  tokenState.expiresAtEpochSeconds = Math.floor(Date.now() / 1000) + (expiresInSeconds > 0 ? expiresInSeconds : 300);
  return { accessToken: tokenState.accessToken, expiresIn: expiresInSeconds };
}

async function resolveEtsToken(providerOrValue) {
  if (!providerOrValue) return "";
  if (typeof providerOrValue === "string") return providerOrValue;
  const resolved = providerOrValue();
  return typeof resolved?.then === "function" ? await resolved : resolved;
}

function base64UrlEncodeFromObject(jsonObject) {
  return base64UrlEncodeBytes(new TextEncoder().encode(JSON.stringify(jsonObject)));
}

function base64UrlEncodeBytes(byteArray) {
  let binaryString = "";
  for (let byteIndex = 0; byteIndex < byteArray.length; byteIndex++) {
    binaryString += String.fromCharCode(byteArray[byteIndex]);
  }
  return btoa(binaryString).replace(/=+$/, "").replace(/\+/g, "-").replace(/\//g, "_");
}

function joinUrl(base, path) {
  if (path.startsWith("http://") || path.startsWith("https://")) return path;
  const left = base.endsWith("/") ? base.slice(0, -1) : base;
  const right = path.startsWith("/") ? path : "/" + path;
  return left + right;
}

async function createDpopJwt({ requestUrl, httpMethod, cryptoKeyPair }) {
  const parsed = new URL(requestUrl);
  const protectedHeader = { typ: "dpop+jwt", alg: "ES256" };
  const publicJwk = await crypto.subtle.exportKey("jwk", cryptoKeyPair.publicKey);
  protectedHeader.jwk = { kty: publicJwk.kty, crv: publicJwk.crv, x: publicJwk.x, y: publicJwk.y };

  const payload = {
    htm: httpMethod,
    htu: parsed.origin + parsed.pathname + parsed.search,
    jti: crypto.randomUUID(),
    iat: Math.floor(Date.now() / 1000)
  };

  const signingInput = base64UrlEncodeFromObject(protectedHeader) + "." + base64UrlEncodeFromObject(payload);
  const signatureDer = await crypto.subtle.sign(
    { name: "ECDSA", hash: "SHA-256" },
    cryptoKeyPair.privateKey,
    new TextEncoder().encode(signingInput)
  );
  const signatureJose = convertDerEcdsaToJoseSignature(new Uint8Array(signatureDer));
  const signatureB64u = base64UrlEncodeBytes(signatureJose);
  return signingInput + "." + signatureB64u;
}

/* Convert DER ECDSA signature to JOSE (r||s), 32 bytes each for P-256 */
function convertDerEcdsaToJoseSignature(derBytes) {
  if (derBytes[0] !== 0x30) throw new Error("Invalid DER signature");
  let bufferOffset = 1;
  let totalLength = derBytes[bufferOffset++];

  if (totalLength & 0x80) {
    const lengthOfLength = totalLength & 0x7f;
    totalLength = 0;
    for (let lengthByteIndex = 0; lengthByteIndex < lengthOfLength; lengthByteIndex++) {
      totalLength = (totalLength << 8) | derBytes[bufferOffset++];
    }
  }

  if (derBytes[bufferOffset++] !== 0x02) throw new Error("Invalid DER signature (no r)");
  let rLength = derBytes[bufferOffset++];
  let rComponent = derBytes.slice(bufferOffset, bufferOffset + rLength);
  bufferOffset += rLength;

  if (derBytes[bufferOffset++] !== 0x02) throw new Error("Invalid DER signature (no s)");
  let sLength = derBytes[bufferOffset++];
  let sComponent = derBytes.slice(bufferOffset, bufferOffset + sLength);

  rComponent = removeLeadingZeroes(rComponent);
  sComponent = removeLeadingZeroes(sComponent);
  if (rComponent.length > 32 || sComponent.length > 32) throw new Error("Invalid ECDSA component length");

  const rPadded = new Uint8Array(32);
  rPadded.set(rComponent, 32 - rComponent.length);
  const sPadded = new Uint8Array(32);
  sPadded.set(sComponent, 32 - sComponent.length);

  const jose = new Uint8Array(64);
  jose.set(rPadded, 0);
  jose.set(sPadded, 32);
  return jose;
}

function removeLeadingZeroes(bytes) {
  let startIndex = 0;
  while (startIndex < bytes.length - 1 && bytes[startIndex] === 0x00) startIndex++;
  return bytes.slice(startIndex);
}
