const DEFAULT_BACKEND_URL = "http://localhost:8080";

export interface ServerConfig {
  backendUrl: string;
}

export function getServerConfig(): ServerConfig {
  return {
    backendUrl: process.env.BACKEND_URL?.trim() || DEFAULT_BACKEND_URL,
  };
}
