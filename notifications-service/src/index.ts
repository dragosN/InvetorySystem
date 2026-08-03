import { startConsumer } from "./consumer";

const shutdown = await startConsumer();

async function stop(signal: string) {
  console.log(JSON.stringify({ msg: "shutting down", signal }));
  await shutdown();
  process.exit(0);
}

process.on("SIGINT", () => void stop("SIGINT"));
process.on("SIGTERM", () => void stop("SIGTERM"));
