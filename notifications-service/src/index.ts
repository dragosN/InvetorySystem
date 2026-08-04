import { startConsumer } from "./consumer";
import { closeRedis } from "./redis";

const shutdownConsumer = await startConsumer();

async function stop(signal: string) {
  console.log(JSON.stringify({ msg: "shutting down", signal }));
  await shutdownConsumer();
  await closeRedis();
  process.exit(0);
}

process.on("SIGINT", () => void stop("SIGINT"));
process.on("SIGTERM", () => void stop("SIGTERM"));
