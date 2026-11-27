package com.github.echoping07.kemaolife.data

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Rule
import org.junit.Test
import org.junit.rules.TemporaryFolder
import java.io.File

/**
 * `AppPreferences` 文件缓存读写、原子替换与旧版 `app_prefs` 键透明迁移单测。
 */
class AppPreferencesCacheFileTest {

    @Rule
    @JvmField
    val tempDir = TemporaryFolder()

    @Test
    fun `全新安装无文件无旧键时返回空串且不落盘`() {
        val file = File(tempDir.root, "timetable_cache.json")
        var removed = false

        val content = AppPreferences.readCacheFile(
            file = file,
            readLegacy = { null },
            removeLegacy = { removed = true },
        )

        assertEquals("", content)
        assertFalse(file.exists())
        assertFalse(removed)
    }

    @Test
    fun `存在旧版偏好键时透明迁移到文件并移除旧键`() {
        val file = File(tempDir.root, "electricity_cache.json")
        val legacyJson = """{"status":{"balance":18.5,"unit":"度"},"usage":{"days":[]}}"""
        var legacy: String? = legacyJson
        var removeCount = 0

        val firstRead = AppPreferences.readCacheFile(
            file = file,
            readLegacy = { legacy },
            removeLegacy = {
                legacy = null
                removeCount++
            },
        )

        assertEquals(legacyJson, firstRead)
        assertTrue(file.isFile)
        assertEquals(legacyJson, file.readText(Charsets.UTF_8))
        assertEquals(1, removeCount)
        assertFalse(File(tempDir.root, "electricity_cache.json.tmp").exists())

        // 迁移完成后直接读文件，不再触碰旧键。
        var legacyReadAgain = false
        val secondRead = AppPreferences.readCacheFile(
            file = file,
            readLegacy = {
                legacyReadAgain = true
                null
            },
            removeLegacy = { removeCount++ },
        )
        assertEquals(legacyJson, secondRead)
        assertFalse(legacyReadAgain)
        assertEquals(1, removeCount)
    }

    @Test
    fun `旧键为空串时仅清理旧键且不创建空文件`() {
        val file = File(tempDir.root, "timetable_cache.json")
        var removed = false

        val content = AppPreferences.readCacheFile(
            file = file,
            readLegacy = { "" },
            removeLegacy = { removed = true },
        )

        assertEquals("", content)
        assertFalse(file.exists())
        assertTrue(removed)
    }

    @Test
    fun `重复写入原子覆盖目标文件且不残留临时文件`() {
        val file = File(tempDir.root, "timetable_cache.json")

        assertTrue(AppPreferences.writeCacheFile(file, """{"version":2,"course":"高等数学"}"""))
        assertEquals("""{"version":2,"course":"高等数学"}""", file.readText(Charsets.UTF_8))
        assertFalse(File(tempDir.root, "timetable_cache.json.tmp").exists())

        assertTrue(AppPreferences.writeCacheFile(file, """{"version":2,"course":"大学物理"}"""))
        assertEquals("""{"version":2,"course":"大学物理"}""", file.readText(Charsets.UTF_8))
        assertFalse(File(tempDir.root, "timetable_cache.json.tmp").exists())
    }

    @Test
    fun `写入空串或显式清理时删除缓存文件与残留临时文件`() {
        val file = File(tempDir.root, "electricity_cache.json")
        val tmp = File(tempDir.root, "electricity_cache.json.tmp")
        assertTrue(AppPreferences.writeCacheFile(file, """{"status":{},"usage":{}}"""))
        tmp.writeText("stale", Charsets.UTF_8)

        assertTrue(AppPreferences.writeCacheFile(file, ""))
        assertFalse(file.exists())
        assertFalse(tmp.exists())

        assertTrue(AppPreferences.writeCacheFile(file, """{"status":{},"usage":{}}"""))
        tmp.writeText("stale", Charsets.UTF_8)
        AppPreferences.deleteCacheFile(file)
        assertFalse(file.exists())
        assertFalse(tmp.exists())
    }
}
