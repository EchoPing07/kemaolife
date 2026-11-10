package com.github.echoping07.kemaolife.ui

import com.github.echoping07.kemaolife.data.Entry
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 课表网格排版的纯函数测试。
 *
 * 这里只验证「小节号 → 行区间 / 并发分栏」的换算；周次、均量这类业务计算
 * 全部在 Go 侧 `core`（AGENTS A3），Kotlin 不重复实现。
 */
class CourseLayoutTest {

    private fun entry(name: String, periods: String) = Entry(
        name = name,
        periods = periods,
        time = "08:30~10:00",
        teacher = "任课教师",
        location = "6 号楼-101",
        weeksText = "1-16周",
    )

    @Test
    fun periodTextIsParsedIntoRange() {
        assertEquals(listOf(1, 2), parsePeriodNumbers("1-2"))
        assertEquals(listOf(1, 2), parsePeriodNumbers("1,2"))
        assertEquals(listOf(5, 6, 7, 8), parsePeriodNumbers("5-8"))
        // 越界小节号被钳到网格范围内（宁可画满也不丢课），非法文本丢弃。
        assertEquals((1..TIMETABLE_ROWS).toList(), parsePeriodNumbers("1-99"))
        assertEquals(emptyList<Int>(), parsePeriodNumbers(""))
    }

    @Test
    fun blockSpansPeriodRange() {
        val blocks = layoutCourseBlocks(listOf(entry("园艺", "1-2"), entry("体育", "5-6")))
        assertEquals(2, blocks.size)
        assertEquals(0, blocks[0].startRow)
        assertEquals(2, blocks[0].spanRows)
        assertEquals(4, blocks[1].startRow)
        assertEquals(2, blocks[1].spanRows)
        assertEquals(1, blocks[0].lanes)
    }

    @Test
    fun overlappingCoursesShareLanes() {
        val blocks = layoutCourseBlocks(
            listOf(entry("A课程", "1-4"), entry("B课程", "3-4"), entry("C课程", "5-6")),
        )
        // 前两门在 3-4 节撞课 → 同一簇、两条 lane；第三门不属于该簇。
        assertEquals(2, blocks[0].lanes)
        assertEquals(2, blocks[1].lanes)
        assertEquals(0, blocks[0].lane)
        assertEquals(1, blocks[1].lane)
        assertEquals(1, blocks[2].lanes)
        assertEquals(0, blocks[2].lane)
    }

    @Test
    fun unresolvablePeriodsAreDroppedFromGrid() {
        val blocks = layoutCourseBlocks(listOf(entry("无节次", ""), entry("正常", "1-2")))
        assertEquals(1, blocks.size)
        assertEquals("正常", blocks[0].entry.name)
    }

    @Test
    fun rowsNeverExceedTimetableHeight() {
        val blocks = layoutCourseBlocks(listOf(entry("很长的课", "1-12")))
        assertEquals(1, blocks.size)
        assertEquals(TIMETABLE_ROWS, blocks[0].startRow + blocks[0].spanRows)
    }

    @Test
    fun dateAndStampFormatting() {
        assertEquals("2026/11/7", prettyDate("2026-11-07"))
        assertEquals("2026/1/5", prettyDate("2026-01-05"))
        // 非 ISO 文本原样返回，不吞内容。
        assertEquals("未知", prettyDate("未知"))
        assertEquals("11/07 21:42", shortStamp("2026-11-07T21:42:07+08:00"))
        assertTrue(shortStamp("").isEmpty())
    }
}
