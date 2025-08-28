"""
Configuration management for Canopy Plugin Python implementation.

This module provides type-safe configuration handling with validation.
"""

import json
from pathlib import Path
from typing import Union, Dict, Any
from dataclasses import dataclass, asdict


@dataclass
class Config:
    """Configuration management for Canopy plugin."""

    chain_id: int = 1
    data_dir_path: str = "/tmp/plugin/"

    def __post_init__(self) -> None:
        """Validate configuration after initialization."""
        if not isinstance(self.chain_id, int) or self.chain_id < 1:
            raise ValueError(
                f"Invalid chain_id: {self.chain_id}. Must be a positive integer."
            )

        if not isinstance(self.data_dir_path, str) or not self.data_dir_path.strip():
            raise ValueError(
                f"Invalid data_dir_path: {self.data_dir_path}. Must be a non-empty string."
            )

    @classmethod
    def from_file(cls, filepath: str) -> "Config":
        """Load configuration from JSON file."""
        if not filepath or not filepath.strip():
            raise ValueError("Filepath must be a non-empty string")

        try:
            config_data = json.loads(Path(filepath).read_text(encoding="utf-8"))

            return cls(
                chain_id=config_data.get("chainId", cls().chain_id),
                data_dir_path=config_data.get("dataDirPath", cls().data_dir_path),
            )
        except (OSError, json.JSONDecodeError) as err:
            raise ValueError(f"Failed to load config from {filepath}: {err}") from err

    def save_to_file(self, filepath: str) -> None:
        """Save configuration to JSON file."""
        if not filepath or not filepath.strip():
            raise ValueError("Filepath must be a non-empty string")

        config_data = {"chainId": self.chain_id, "dataDirPath": self.data_dir_path}

        try:
            file_path = Path(filepath)
            file_path.parent.mkdir(parents=True, exist_ok=True)
            file_path.write_text(
                json.dumps(config_data, indent=2, ensure_ascii=False), encoding="utf-8"
            )
        except OSError as err:
            raise ValueError(f"Failed to save config to {filepath}: {err}") from err

    def update(self, **kwargs: Union[str, int]) -> "Config":
        """Create a copy with updated values."""
        current = asdict(self)
        current.update(kwargs)
        return Config(**current)

    def to_dict(self) -> Dict[str, Any]:
        """Convert to dict for serialization."""
        return {"chainId": self.chain_id, "dataDirPath": self.data_dir_path}
