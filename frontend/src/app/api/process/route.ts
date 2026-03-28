export const dynamic = "force-dynamic";

const BACKEND_URL = process.env.BACKEND_URL || "http://localhost:8080";

export async function POST(req: Request) {
  try {
    const formData = await req.formData();
    const url = new URL(req.url);
    const isStream = url.searchParams.get("stream") === "true";

    const backendUrl = new URL("/process", BACKEND_URL);
    if (isStream) {
      backendUrl.searchParams.set("stream", "true");
    }

    const backendResponse = await fetch(backendUrl.toString(), {
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

  try {
    const backendUrl = new URL(path, BACKEND_URL);
    const response = await fetch(backendUrl.toString());
    return response;
  } catch (error) {
    return new Response(`Backend unreachable`, { status: 503 });
  }
}
