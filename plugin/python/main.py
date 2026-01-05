"""Main entry point for the Canopy blockchain plugin."""

import asyncio
import signal
import logging

from contract import start_plugin, default_config

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def main() -> None:
    """Start the plugin and wait for shutdown signal."""
    logger.info("Starting Canopy Plugin")

    # Start the plugin
    plugin = await start_plugin(default_config())

    # Wait for shutdown signal
    stop = asyncio.Event()
    loop = asyncio.get_running_loop()
    loop.add_signal_handler(signal.SIGINT, stop.set)
    loop.add_signal_handler(signal.SIGTERM, stop.set)

    await stop.wait()

    # Graceful shutdown
    logger.info("Shutting down plugin...")
    await plugin.close()


if __name__ == "__main__":
    asyncio.run(main())
