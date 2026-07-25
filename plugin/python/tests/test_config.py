import tempfile
import json
import pytest
from pathlib import Path

from contract import Config, default_config, new_config_from_file

class TestConfig:
    def test_default_config(self):
        config = Config()
        assert config.chain_id == 1
        assert config.data_dir_path == "/tmp/plugin/"

    def test_custom_config(self):
        config = Config(chain_id=42, data_dir_path="/custom/path/")
        assert config.chain_id == 42
        assert config.data_dir_path == "/custom/path/"

    def test_config_validation_invalid_chain_id(self):
        with pytest.raises(ValueError, match="Invalid chain_id"):
            Config(chain_id=0)

    def test_config_validation_invalid_data_dir(self):
        with pytest.raises(ValueError, match="Invalid data_dir_path"):
            Config(data_dir_path="")

    def test_default_config_fn(self):
        config = default_config()
        assert config.chain_id == 1
        assert config.data_dir_path == "/tmp/plugin/"

    def test_new_config_from_file(self):
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
            json.dump({'chainId': 99, 'dataDirPath': '/test/path/'}, f)
            temp_path = f.name
        try:
            config = new_config_from_file(temp_path)
            assert config.chain_id == 99
            assert config.data_dir_path == "/test/path/"
        finally:
            Path(temp_path).unlink(missing_ok=True)

    def test_new_config_from_file_missing_fields(self):
        with tempfile.NamedTemporaryFile(mode='w', suffix='.json', delete=False) as f:
            json.dump({'chainId': 99}, f)
            temp_path = f.name
        try:
            config = new_config_from_file(temp_path)
            assert config.chain_id == 99
            assert config.data_dir_path == "/tmp/plugin/"
        finally:
            Path(temp_path).unlink(missing_ok=True)

    def test_new_config_from_file_invalid(self):
        with pytest.raises(ValueError, match="Failed to load config"):
            new_config_from_file("/non/existent/file.json")