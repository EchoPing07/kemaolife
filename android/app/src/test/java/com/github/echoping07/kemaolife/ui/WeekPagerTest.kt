package com.github.echoping07.kemaolife.ui

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 课表横向翻页（docs/002 §5.3 tab1）的手感契约：
 *  - 翻页序列覆盖整个学期，锚点周在学期外也要能落页；
 *  - `lastWeek` 未知时不能把翻页功能悄悄关掉（否则用户看到「划不动」）；
 *  - 邻居页必须能被预取到，否则拖动时是空白页。
 *
 * 周次本身怎么算在 Go 侧（AGENTS A3），这里只验证序列与索引换算。
 */
class WeekPagerTest {

    @Test
    fun `学期已知时序列覆盖 1 到 lastWeek`() {
        assertEquals((1..18).toList(), WeekPager.pages(lastWeek = 18, anchor = 7))
    }

    @Test
    fun `锚点周在学期外也要并进序列`() {
        // 假期查询会返回 lastWeek + 1，若直接按 1..lastWeek 建序列，
        // 首帧找不到当前周就会被弹回第 1 周。
        val pages = WeekPager.pages(lastWeek = 18, anchor = 19)
        assertEquals(19, pages.last())
        assertEquals(19, pages[WeekPager.indexOf(pages, 19)])
    }

    @Test
    fun `lastWeek 未知时仍可翻页而不是卡住`() {
        val pages = WeekPager.pages(lastWeek = 0, anchor = 10)
        assertTrue(pages.size > 1)
        assertEquals(10, pages[WeekPager.indexOf(pages, 10)])
        // 窗口两侧都要能翻，不能只给当前周。
        assertTrue(pages.indexOf(9) >= 0)
        assertTrue(pages.indexOf(11) >= 0)
    }

    @Test
    fun `未知周次落到第 0 页而不是抛异常`() {
        val pages = WeekPager.pages(lastWeek = 18, anchor = 7)
        assertEquals(0, WeekPager.indexOf(pages, 999))
        assertEquals(0, WeekPager.indexOf(emptyList(), 7))
    }

    @Test
    fun `邻居周按学期范围裁剪`() {
        assertEquals(listOf(2), WeekPager.neighbors(week = 1, lastWeek = 18))
        assertEquals(listOf(17), WeekPager.neighbors(week = 18, lastWeek = 18))
        assertEquals(listOf(4, 6), WeekPager.neighbors(week = 5, lastWeek = 18))
        // lastWeek 未知时不裁剪（越界周次由 Go 返回 inSemester:false）。
        assertEquals(listOf(9, 11), WeekPager.neighbors(week = 10, lastWeek = 0))
        assertEquals(emptyList<Int>(), WeekPager.neighbors(week = 0, lastWeek = 18))
    }

    @Test
    fun `邻页缓存只保留当前周附近且一定含当前周`() {
        val kept = WeekPager.retained(listOf(1, 4, 5, 6, 9), current = 5)
        assertEquals(setOf(4, 5, 6), kept)
        // 当前周即使超出窗口也必须留下：滑回来不能闪空白。
        assertEquals(setOf(5), WeekPager.retained(listOf(5), current = 5))
    }
}
