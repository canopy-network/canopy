"""
Main entry point for the Canopy blockchain plugin.

Starts the plugin with Unix socket communication and handles graceful shutdown.
"""

import asyncio
import signal
import sys
import os
import logging
from typing import Optional

from plugin.config import Config
from plugin.socket import SocketClient, SocketClientOptions


# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class PluginApp:
    """Main plugin application with graceful shutdown handling."""
    
    def __init__(self) -> None:
        self.socket_client: Optional[SocketClient] = None
        self._shutdown_event = asyncio.Event()
    
    async def start(self) -> None:
        """Start the plugin application."""
        try:
            logger.info('Starting Canopy Plugin')
            
            # Create default configuration
            config = Config()
            
            # Log configuration
            logger.info(f'Plugin configuration:')
            logger.info(f'  - Chain ID: {config.chain_id}')
            logger.info(f'  - Data Directory: {config.data_dir_path}')
            logger.info(f'  - Socket Path: {os.path.join(config.data_dir_path, "plugin.sock")}')
            
            # Create socket client options
            options = SocketClientOptions(config=config)
            
            # Start the socket client
            self.socket_client = SocketClient(options)
            await self.socket_client.start()
            
            logger.info('Plugin started successfully - waiting for FSM requests...')
            
            # Setup signal handlers for graceful shutdown
            self._setup_signal_handlers()
            
            # Keep the process alive until shutdown
            await self._shutdown_event.wait()
            
            # Perform graceful shutdown
            await self.shutdown()
        
        except Exception as error:
            logger.error(f'Failed to start plugin: {error}')
            sys.exit(1)
    
    async def shutdown(self) -> None:
        """Shutdown the plugin gracefully."""
        logger.info('Received shutdown signal, closing plugin...')
        
        try:
            if self.socket_client:
                await asyncio.wait_for(
                    self.socket_client.close(),
                    timeout=10.0  # 10 second timeout
                )
            
            logger.info('Plugin shut down gracefully')
        
        except asyncio.TimeoutError:
            logger.error('Shutdown timeout exceeded')
            sys.exit(1)
        except Exception as error:
            logger.error(f'Error during shutdown: {error}')
            sys.exit(1)
    
    def _setup_signal_handlers(self) -> None:
        """Setup signal handlers for graceful shutdown."""
        loop = asyncio.get_running_loop()
        
        def signal_handler() -> None:
            logger.info('Received shutdown signal')
            self._shutdown_event.set()
        
        loop.add_signal_handler(signal.SIGINT, signal_handler)
        loop.add_signal_handler(signal.SIGTERM, signal_handler)


async def main() -> None:
    """Main entry point for the plugin."""
    app = PluginApp()
    await app.start()


def run() -> None:
    """Run the main application with asyncio."""
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info('Plugin interrupted by user')
    except Exception as error:
        logger.error(f'Fatal error: {error}')
        sys.exit(1)


if __name__ == '__main__':
    run()