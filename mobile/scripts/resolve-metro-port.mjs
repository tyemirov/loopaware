import net from "node:net";

const defaultStartPort = 8081;
const defaultEndPort = 8099;
const loopbackHost = "127.0.0.1";

const requestedPort = Number.parseInt(process.env.LOOPAWARE_MOBILE_METRO_PORT ?? "", 10);
const startPort = Number.parseInt(process.env.LOOPAWARE_MOBILE_METRO_PORT_START ?? "", 10) || defaultStartPort;
const endPort = Number.parseInt(process.env.LOOPAWARE_MOBILE_METRO_PORT_END ?? "", 10) || defaultEndPort;

if (Number.isInteger(requestedPort) && requestedPort > 0) {
  await assertPortAvailable(requestedPort);
  console.log(String(requestedPort));
  process.exit(0);
}

for (let candidatePort = startPort; candidatePort <= endPort; candidatePort += 1) {
  if (await isPortAvailable(candidatePort)) {
    console.log(String(candidatePort));
    process.exit(0);
  }
}

throw new Error(`loopaware_mobile_metro_port_unavailable: ${startPort}-${endPort}`);

async function assertPortAvailable(port) {
  if (await isPortAvailable(port)) {
    return;
  }
  throw new Error(`loopaware_mobile_metro_port_in_use: ${port}`);
}

async function isPortAvailable(port) {
  return (await canListenOnPort(port)) && (await canListenOnPort(port, loopbackHost));
}

function canListenOnPort(port, host) {
  return new Promise((resolve) => {
    const server = net.createServer();
    server.once("error", () => resolve(false));
    server.once("listening", () => {
      server.close(() => resolve(true));
    });
    server.listen(typeof host === "string" ? { port, host } : { port });
  });
}
