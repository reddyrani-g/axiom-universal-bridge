export const dynamic = "force-dynamic";

import { getServerConfig } from "@/lib/server-config";

export async function POST(req: Request) {
  try {
    const formData = await req.formData();
    const url = new URL(req.url);
    const isStream = url.searchParams.get("stream") === "true";
    const { backendUrl } = getServerConfig();

    const targetUrl = new URL("/process", backendUrl);
    if (isStream) {
      targetUrl.searchParams.set("stream", "true");
    }

    const backendResponse = await fetch(targetUrl.toString(), {
      method: "POST",
      body: formData,
    });

    if (!backendResponse.ok) {
      return new Response(`Backend error: ${backendResponse.statusText}`, {
        status: backendResponse.status,
      });
    }

    // If streaming, return the body as-is for SSE
    if (isStream) {
      return new Response(backendResponse.body, {
        headers: {
          "Content-Type": "text/event-stream",
          "Cache-Control": "no-cache",
          "Connection": "keep-alive",
        },
      });
    }

    // Otherwise return JSON response
    return backendResponse;
  } catch (error) {
    console.error("API route error:", error);
    return new Response(`API error: ${error instanceof Error ? error.message : "Unknown error"}`, {
      status: 500,
    });
  }
}

export async function GET(req: Request) {
  const url = new URL(req.url);
  const path = url.searchParams.get("path") || "/health";
  const { backendUrl } = getServerConfig();

  try {
    const targetUrl = new URL(path, backendUrl);
    const response = await fetch(targetUrl.toString());
    return response;
  } catch (error) {
    return new Response(`Backend unreachable`, { status: 503 });
  }
}
