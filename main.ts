// src/index.ts
import { Hono } from "hono";

const app = new Hono();

// Config
const API_KEYS = Deno.env.get("API_KEYS")?.split(",") ?? [];

if (!API_KEYS.length) {
  throw new Error("API_KEYS environment variable not set or empty");
}

let currentKeyIndex = 0;

function rotateAPIKey() {
  const prevIndex = currentKeyIndex;
  currentKeyIndex = (currentKeyIndex + 1) % API_KEYS.length;
  console.log(`Rotated API key from index ${prevIndex} to ${currentKeyIndex}`);
}

app.all("/*", async (c) => {
  const path = c.req.path;
  const method = c.req.method;
  const headers = c.req.header();

  // Try each API key
  for (let i = 0; i < API_KEYS.length; i++) {
    const apiKey = API_KEYS[currentKeyIndex];

    try {
      const response = await fetch(`https://api.anthropic.com${path}`, {
        method,
        headers: {
          ...headers,
          "x-api-key": apiKey,
        },
        body: method !== "GET" ? await c.req.blob() : undefined,
      });

      if (response.status === 429) {
        rotateAPIKey();
        console.log("Rate limited, rotating API key");
        continue;
      }

      // Stream the response
      return new Response(response.body, {
        status: response.status,
        headers: response.headers,
      });
    } catch (error) {
      console.error(`Request failed with API key ${currentKeyIndex}:`, error);
      rotateAPIKey();
    }
  }

  return new Response("All API keys exhausted", { status: 500 });
});

Deno.serve({ port: parseInt(Deno.env.get("PORT") || "8080") }, app.fetch);
