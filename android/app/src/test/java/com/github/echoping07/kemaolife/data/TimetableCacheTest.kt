package com.github.echoping07.kemaolife.data

import org.json.JSONArray
import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 冷启动首帧用的课表缓存（docs/002 §5.3「每次打开」）：
 * Kotlin 侧与电费缓存同口径，只写 UI 必需字段，不含 Cookie / 凭据。
 */
class TimetableCacheTest {

    private val week = WeekSchedule(
        week = 7,
        days = listOf(
            DaySchedule(
                date = "2026-11-09",
                week = 7,
                weekday = 1,
                weekdayCn = "一",
                inSemester = true,
                entries = listOf(
                    Entry(
                        name = "高等数学",
                        periods = "1-2",
                        time = "08:00~09:40",
                        teacher = "<教师>",
                        location = "<校区>_<楼栋>_<楼层>_<房间>",
                        weeksText = "1-16",
                    ),
                ),
            ),
        ),
    )

    @Test
    fun `缓存往返保留周次日次与数据时间`() {
        val cached = TimetableCache.decode(TimetableCache.encode(week, "2026-11-08T09:00:00+08:00"))

        assertEquals(7, cached?.schedule?.week)
        assertEquals("2026-11-09", cached?.schedule?.days?.single()?.date)
        assertEquals("高等数学", cached?.schedule?.days?.single()?.entries?.single()?.name)
        assertEquals("2026-11-08T09:00:00+08:00", cached?.syncedAt)
        assertTrue(TimetableCache.encode(week, "2026-11-08T09:00:00+08:00").contains("\"version\":${TimetableCache.VERSION}"))
    }

    @Test
    fun `旧版本与缺版本的展示镜像必须失效`() {
        val raw = TimetableCache.encode(week, "2026-11-08T09:00:00+08:00")
        assertNull(TimetableCache.decode(raw.replace("\"version\":${TimetableCache.VERSION}", "\"version\":1")))
        // 用非空 days 才能证明失效来自缺 version，而不是空数组分支。
        val legacy = JSONObject()
            .put("week", 7)
            .put("syncedAt", "2026-11-08T09:00:00+08:00")
            .put("days", JSONArray().put(JSONObject().put("date", "2026-11-09")))
        assertNull(TimetableCache.decode(legacy.toString()))
    }

    @Test
    fun `空串与损坏内容都视为无缓存而不是抛异常`() {
        assertNull(TimetableCache.decode(""))
        assertNull(TimetableCache.decode("{"))
        assertNull(TimetableCache.decode("{}"))
        assertNull(TimetableCache.decode("{\"days\":[]}"))
    }

    @Test
    fun `缓存不含上游快照的敏感字段且缺字段有零值语义`() {
        val raw = TimetableCache.encode(week, "2026-11-08T09:00:00+08:00")
        assertFalse(raw.contains("cookie"))
        assertFalse(raw.contains("Cookie"))
        assertFalse(raw.contains("studentId"))
        assertFalse(raw.contains("roomVerify"))

        val sparse = TimetableCache.decode(
            JSONObject()
                .put("version", TimetableCache.VERSION)
                .put("week", 3)
                .put("syncedAt", "")
                .put("days", JSONArray().put(JSONObject().put("date", "2026-10-05")))
                .toString(),
        )
        assertEquals(3, sparse?.schedule?.week)
        assertEquals("", sparse?.syncedAt)
        val sparseDay = sparse?.schedule?.days?.single()
        assertEquals("2026-10-05", sparseDay?.date)
        assertEquals(0, sparseDay?.week)
        assertFalse(sparseDay?.inSemester ?: true)
        assertTrue(sparseDay?.entries?.isEmpty() ?: false)
    }
}
