"""
Unit tests for Config class.

Tests configuration management, validation, and file operations.
"""

import tempfile
import json
import pytest
from pathlib import Path

from plugin.config import Config, ConfigOptions


class TestConfig:
    """Test cases for Config class."""
    
    def test_default_config(self):
        """Test default configuration creation."""
        config = Config()
        
        assert config.chain_id == 1
        assert config.data_dir_path == "/tmp/plugin/"
    
    def test_custom_config(self):
        """Test configuration with custom options."""
        options = ConfigOptions(chain_id=42, data_dir_path="/custom/path/")
        config = Config(options)
        
        assert config.chain_id == 42
        assert config.data_dir_path == "/custom/path/"
    
    def test_config_validation_invalid_chain_id(self):
        """Test configuration validation with invalid chain ID."""
        with pytest.raises(ValueError, match="Invalid chain_id"):
            Config(ConfigOptions(chain_id=0))  # Must be positive
    
    def test_config_validation_invalid_data_dir(self):
        """Test configuration validation with invalid data directory."""
        with pytest.raises(ValueError, match="Invalid data_dir_path"):
            Config(ConfigOptions(data_dir_path=""))  # Must be non-empty
    
    def test_config_update(self):
        """Test configuration update method."""
        config = Config()
        updated_config = config.update(chain_id=99, data_dir_path="/new/path/")
        
        # Original should be unchanged
        assert config.chain_id == 1
        assert config.data_dir_path == "/tmp/plugin/"
        
        # Updated should have new values
        assert updated_config.chain_id == 99
        assert updated_config.data_dir_path == "/new/path/"
    
    def test_config_to_dict(self):
        """Test configuration serialization to dict."""
        config = Config()
        data = config.to_dict()
        
        expected = {
            'chainId': 1,
            'dataDirPath': '/tmp/plugin/'
        }
        assert data == expected
    
    def test_config_str_representation(self):
        """Test string representation of configuration."""
        config = Config()
        str_repr = str(config)
        
        assert 'Config(' in str_repr
        assert 'chain_id=1' in str_repr
        assert 'data_dir_path="/tmp/plugin/"' in str_repr
    
    def test_config_equality(self):
        """Test configuration equality comparison."""
        config1 = Config()
        config2 = Config()
        config3 = Config(ConfigOptions(chain_id=2))
        
        assert config1 == config2
        assert config1 != config3
        assert config1 != "not a config"
    
    def test_save_and_load_config_sync(self):
        """Test synchronous save and load operations."""
        config = Config(ConfigOptions(chain_id=42, data_dir_path="/test/path/"))
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
            temp_path = f.name
        
        try:
            # Save config
            config.save_to_file_sync(temp_path)
            
            # Load config
            loaded_config = Config.from_file_sync(temp_path)
            
            assert loaded_config.chain_id == 42
            assert loaded_config.data_dir_path == "/test/path/"
            
        finally:
            Path(temp_path).unlink(missing_ok=True)
    
    def test_load_config_with_missing_fields(self):
        """Test loading config with missing fields uses defaults."""
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
            json.dump({'chainId': 99}, f)
            temp_path = f.name
        
        try:
            loaded_config = Config.from_file_sync(temp_path)
            
            assert loaded_config.chain_id == 99
            assert loaded_config.data_dir_path == "/tmp/plugin/"  # Default value
            
        finally:
            Path(temp_path).unlink(missing_ok=True)
    
    def test_load_config_invalid_file(self):
        """Test loading config from non-existent file."""
        with pytest.raises(ValueError, match="Failed to load config"):
            Config.from_file_sync("/non/existent/file.json")
    
    def test_save_config_invalid_path(self):
        """Test saving config to invalid path."""
        config = Config()
        
        with pytest.raises(ValueError, match="Filepath must be a non-empty string"):
            config.save_to_file_sync("")


@pytest.mark.asyncio
class TestConfigAsync:
    """Async test cases for Config class."""
    
    async def test_save_and_load_config_async(self):
        """Test asynchronous save and load operations."""
        config = Config(ConfigOptions(chain_id=123, data_dir_path="/async/test/"))
        
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
            temp_path = f.name
        
        try:
            # Save config asynchronously
            await config.save_to_file(temp_path)
            
            # Load config asynchronously
            loaded_config = await Config.from_file(temp_path)
            
            assert loaded_config.chain_id == 123
            assert loaded_config.data_dir_path == "/async/test/"
            
        finally:
            Path(temp_path).unlink(missing_ok=True)