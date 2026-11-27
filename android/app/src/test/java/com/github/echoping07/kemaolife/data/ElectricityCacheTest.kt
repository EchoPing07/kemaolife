package com.github.echoping07.kemaolife.data

import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 冷启动首帧用的电费缓存（docs/android.md §2 / §7）：
 * 按字段白名单编码，绝不持久化 `roomName` / `roomVerify` / 学号 / Cookie（P1 / P2），
 * 并与历史缓存 JSON 格式完全兼容。
 */
class ElectricityCacheTest {

    private val sampleReport = ElectricityReport(
        status = ElectricityStatus(
            roomName = "<校区>_<楼栋>_<楼层>_<房间>",
            balance = 35.99,
            unit = "度",
            collectedAt = "2026-11-04T22:41:37+08:00",
            dailyAvg = 6.25,
            estimatedDays = 5,
        ),
        usage = ElectricityUsage(
            yearMonth = "2026-11",
            monthUse = 63.13,
            dataUntil = "2026-11-04",
            days = listOf(
                DayUsage(date = "2026-11-03", use = 11.64),
                DayUsage(date = "2026-11-04", use = 11.75),
            ),
        ),
    )

    @Test
    fun `缓存往返保留余额均量可用天数与每日用量`() {
        val raw = ElectricityCache.encode(sampleReport)
        val decoded = requireNotNull(ElectricityCache.decode(raw))

        assertEquals(35.99, decoded.status.balance, 1e-9)
        assertEquals("度", decoded.status.unit)
        assertEquals("2026-11-04T22:41:37+08:00", decoded.status.collectedAt)
        assertEquals(6.25, requireNotNull(decoded.status.dailyAvg), 1e-9)
        assertEquals(5, decoded.status.estimatedDays)
        assertEquals("2026-11", decoded.usage.yearMonth)
        assertEquals(63.13, decoded.usage.monthUse, 1e-9)
        assertEquals("2026-11-04", decoded.usage.dataUntil)
        assertEquals(2, decoded.usage.days.size)
        assertEquals("2026-11-03", decoded.usage.days[0].date)
        assertEquals(11.64, decoded.usage.days[0].use, 1e-9)
        assertEquals("2026-11-04", decoded.usage.days[1].date)
        assertEquals(11.75, decoded.usage.days[1].use, 1e-9)
    }

    @Test
    fun `dailyAvg 与 estimatedDays 为 null 时往返保持 null 而非 0`() {
        val report = sampleReport.copy(
            status = sampleReport.status.copy(dailyAvg = null, estimatedDays = null),
        )
        val decoded = requireNotNull(ElectricityCache.decode(ElectricityCache.encode(report)))

        assertNull(decoded.status.dailyAvg)
        assertNull(decoded.status.estimatedDays)
    }

    @Test
    fun `绝不持久化宿舍房间号或凭据字段且解码时剥离脏 roomName`() {
        val raw = ElectricityCache.encode(sampleReport)

        assertFalse(raw.contains("roomName"))
        assertFalse(raw.contains("<校区>"))
        assertFalse(raw.contains("roomVerify"))
        assertFalse(raw.contains("studentId"))
        assertFalse(raw.contains("cookie", ignoreCase = true))
        assertFalse(raw.contains("password", ignoreCase = true))

        val decoded = requireNotNull(ElectricityCache.decode(raw))
        assertEquals("", decoded.status.roomName)

        // 即使历史或外部脏数据混入 roomName，解码也必须剥离。
        val dirty = JSONObject(raw).apply {
            getJSONObject("status").put("roomName", "<校区>_<楼栋>_<楼层>_<房间>")
        }
        val cleaned = requireNotNull(ElectricityCache.decode(dirty.toString()))
        assertEquals("", cleaned.status.roomName)
    }

    @Test
    fun `兼容旧版 AppViewModel 写入的电费缓存 JSON`() {
        val legacyJson = """
            {
              "status": {
                "balance": 12.5,
                "unit": "度",
                "collectedAt": "2026-11-04 22:41",
                "dailyAvg": null,
                "estimatedDays": 3
              },
              "usage": {
                "yearMonth": "2026-11",
                "monthUse": 20.0,
                "dataUntil": "2026-11-04",
                "days": [{"date": "2026-11-04", "use": 4.2}]
              }
            }
        """.trimIndent()

        val decoded = requireNotNull(ElectricityCache.decode(legacyJson))
        assertEquals(12.5, decoded.status.balance, 1e-9)
        assertNull(decoded.status.dailyAvg)
        assertEquals(3, decoded.status.estimatedDays)
        assertEquals(1, decoded.usage.days.size)
        assertEquals(4.2, decoded.usage.days.single().use, 1e-9)
    }

    @Test
    fun `空串损坏内容或缺段 JSON 返回 null 而不抛异常`() {
        assertNull(ElectricityCache.decode(""))
        assertNull(ElectricityCache.decode("   "))
        assertNull(ElectricityCache.decode("{"))
        assertNull(ElectricityCache.decode("{}"))
        assertNull(ElectricityCache.decode("""{"status":{"balance":1.0}}"""))
        assertNull(ElectricityCache.decode("""{"usage":{"yearMonth":"2026-11"}}"""))
        assertTrue(ElectricityCache.decode("""{"status":{},"usage":{}}""") != null)
    }
}
