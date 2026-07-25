"""
Unit tests for the plugin Config dataclass and helpers.

Covers the current `contract.plugin` API: the `Config` dataclass, `default_config()`,
and `new_config_from_file()`.
"""

import json
import tempfile
from pathlib import Path

import pytest

from contract.plugin import Config, default_config, new_config_from_file


class TestConfig:
    """Test cases for the Config dataclass."""

    def test_default_values(self):
        """A Config built with no args uses the documented defaults."""
        config = Config()

        assert config.chain_id == 1
        assert config.data_dir_path == "/tmp/plugin/"
        assert config.rpc_address == "0.0.0.0:50010"

    def test_custom_values(self):
        """Custom values are stored as provided."""
        config = Config(
            chain_id=42,
            data_dir_path="/custom/path/",
            rpc_address="127.0.0.1:9000",
        )

        assert config.chain_id == 42
        assert config.data_dir_path == "/custom/path/"
        assert config.rpc_address == "127.0.0.1:9000"

    def test_invalid_chain_id(self):
        """chain_id must be a positive integer."""
        with pytest.raises(ValueError, match="Invalid chain_id"):
            Config(chain_id=0)

    def test_invalid_data_dir(self):
        """data_dir_path must be a non-empty string."""
        with pytest.raises(ValueError, match="Invalid data_dir_path"):
            Config(data_dir_path="")

    def test_default_config_helper(self):
        """default_config() returns the canonical default Config."""
        config = default_config()

        assert config.chain_id == 1
        assert config.data_dir_path == "/tmp/plugin/"
        assert config.rpc_address == "0.0.0.0:50010"


class TestConfigFromFile:
    """Test cases for loading configuration from a JSON file."""

    def test_load_full_config(self):
        """All fields present in the file are loaded."""
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump(
                {
                    "chainId": 42,
                    "dataDirPath": "/test/path/",
                    "rpcAddress": "127.0.0.1:1234",
                },
                f,
            )
            temp_path = f.name

        try:
            config = new_config_from_file(temp_path)

            assert config.chain_id == 42
            assert config.data_dir_path == "/test/path/"
            assert config.rpc_address == "127.0.0.1:1234"
        finally:
            Path(temp_path).unlink(missing_ok=True)

    def test_load_with_missing_fields_uses_defaults(self):
        """Missing fields fall back to defaults."""
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump({"chainId": 99}, f)
            temp_path = f.name

        try:
            config = new_config_from_file(temp_path)

            assert config.chain_id == 99
            assert config.data_dir_path == "/tmp/plugin/"
            assert config.rpc_address == "0.0.0.0:50010"
        finally:
            Path(temp_path).unlink(missing_ok=True)

    def test_load_missing_file_raises(self):
        """Loading a non-existent file raises a descriptive ValueError."""
        with pytest.raises(ValueError, match="Failed to load config"):
            new_config_from_file("/non/existent/file.json")
