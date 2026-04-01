const PREFIX = "clier";

export const logger = {
  debug(message: string, context?: Record<string, unknown>) {
    console.debug(`[${PREFIX}] ${message}`, context ?? "");
  },
  error(message: string, context?: Record<string, unknown>) {
    console.error(`[${PREFIX}] ${message}`, context ?? "");
  },
  warn(message: string, context?: Record<string, unknown>) {
    console.warn(`[${PREFIX}] ${message}`, context ?? "");
  },
};
