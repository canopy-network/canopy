package com.canopy.plugin.config

import kotlinx.coroutines.runBlocking
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.assertThrows
import java.io.File
import kotlin.test.BeforeTest

class ConfigTest {
    @BeforeTest
    fun setup() {
        // Clean up any test files
        File("/tmp/test-config.json").delete()
    }

    @Test
    fun `should create config with default values`() {
        val config = Config.defaultConfig()

        assertEquals(1, config.chainId)
        assertEquals("/tmp/plugin/", config.dataDirPath)
    }

    @Test
    fun `should create config with custom values`() {
        val config =
            Config(
                ConfigOptions(
                    chainId = 42,
                    dataDirPath = "/custom/path/",
                ),
            )

        assertEquals(42, config.chainId)
        assertEquals("/custom/path/", config.dataDirPath)
    }

    @Test
    fun `should validate chainId must be positive`() {
        assertThrows<IllegalArgumentException> {
            Config(
                ConfigOptions(
                    chainId = 0,
                    dataDirPath = "/tmp/",
                ),
            )
        }

        assertThrows<IllegalArgumentException> {
            Config(
                ConfigOptions(
                    chainId = -1,
                    dataDirPath = "/tmp/",
                ),
            )
        }
    }

    @Test
    fun `should validate dataDirPath is not empty`() {
        assertThrows<IllegalArgumentException> {
            Config(
                ConfigOptions(
                    chainId = 1,
                    dataDirPath = "",
                ),
            )
        }

        assertThrows<IllegalArgumentException> {
            Config(
                ConfigOptions(
                    chainId = 1,
                    dataDirPath = "   ",
                ),
            )
        }
    }

    @Test
    fun `should save and load config from file`() =
        runBlocking {
            val originalConfig =
                Config(
                    ConfigOptions(
                        chainId = 100,
                        dataDirPath = "/test/data/",
                    ),
                )

            val filepath = "/tmp/test-config.json"
            originalConfig.saveToFile(filepath)

            val loadedConfig = Config.fromFile(filepath)

            assertEquals(originalConfig.chainId, loadedConfig.chainId)
            assertEquals(originalConfig.dataDirPath, loadedConfig.dataDirPath)

            // Clean up
            File(filepath).delete()
        }

    @Test
    fun `should update config with partial values`() {
        val originalConfig =
            Config(
                ConfigOptions(
                    chainId = 1,
                    dataDirPath = "/original/",
                ),
            )

        val updatedConfig =
            originalConfig.update(
                ConfigOptions(chainId = 2),
            )

        assertEquals(2, updatedConfig.chainId)
        assertEquals("/original/", updatedConfig.dataDirPath)
    }

    @Test
    fun `should convert config to JSON`() {
        val config =
            Config(
                ConfigOptions(
                    chainId = 42,
                    dataDirPath = "/test/",
                ),
            )

        val json = config.toJSON()

        assertEquals(42, json.chainId)
        assertEquals("/test/", json.dataDirPath)
    }

    @Test
    fun `should check config equality`() {
        val config1 =
            Config(
                ConfigOptions(
                    chainId = 1,
                    dataDirPath = "/test/",
                ),
            )

        val config2 =
            Config(
                ConfigOptions(
                    chainId = 1,
                    dataDirPath = "/test/",
                ),
            )

        val config3 =
            Config(
                ConfigOptions(
                    chainId = 2,
                    dataDirPath = "/test/",
                ),
            )

        assertTrue(config1 == config2)
        assertFalse(config1 == config3)
    }

    @Test
    fun `should provide string representation`() {
        val config =
            Config(
                ConfigOptions(
                    chainId = 42,
                    dataDirPath = "/test/",
                ),
            )

        val str = config.toString()

        assertTrue(str.contains("42"))
        assertTrue(str.contains("/test/"))
    }
}
