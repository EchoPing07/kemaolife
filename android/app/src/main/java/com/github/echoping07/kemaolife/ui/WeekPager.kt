package com.github.echoping07.kemaolife.ui

import kotlin.math.abs

/**
 * 课表横向翻页的页码 / 邻居周策略（docs/002 §5.3 tab1）。
 *
 * 这里**只做序列与索引换算**：周次本身由 Go 侧算（AGENTS A3），
 * UI 拿到的只是 `timetable.week` 返回的 `week` 与可选 `lastWeek`。
 * 抽成纯函数是因为「翻页边界」「邻居页要预取哪几周」都是手感契约 ——
 * 出错的表现是空白页或翻不回去，而不是编译错误。
 */
internal object WeekPager {

    /**
     * 学期范围未知（`lastWeek` 缺省 / 为 0）时允许翻的窗口半径。
     *
     * 为什么不是「只给当前周」：那等于把翻页功能悄悄关掉，用户看到的是「划不动」。
     * 越界周次由 Go 侧返回 `inSemester:false` 的成功结果，落页显示「不在本学期范围」，
     * 这是诚实的降级，比翻不动好。
     */
    const val FALLBACK_RADIUS = 24

    /** 邻页缓存的保留窗口：两侧各 2 周。 */
    const val WINDOW = 2

    /**
     * 翻页容器的周次序列。
     *
     * `lastWeek > 0` 时是学期 `1..lastWeek`；锚点周（用户当前所在周）若落在学期外
     * （假期查询会返回 `lastWeek + 1`），也要并进序列 —— 否则首帧找不到当前周，
     * pager 会把用户弹到第 1 周。
     */
    fun pages(lastWeek: Int, anchor: Int): List<Int> {
        val base = if (lastWeek >= 1) {
            (1..lastWeek).toList()
        } else {
            val center = if (anchor >= 1) anchor else 1
            (maxOf(1, center - FALLBACK_RADIUS)..(center + FALLBACK_RADIUS)).toList()
        }
        if (anchor < 1 || anchor in base) return base
        return (base + anchor).sorted()
    }

    /** 当前周所在的页下标；不在序列里就落到第 0 页（翻页不能因数据异常崩）。 */
    fun indexOf(pages: List<Int>, week: Int): Int {
        val index = pages.indexOf(week)
        return if (index >= 0) index else 0
    }

    /**
     * 需要预取的邻居周：当周 ±1，按学期范围裁剪。
     *
     * 预取是「拖动时邻居页不空白」的前提：pager 会提前组合相邻页，
     * 没有数据就只能留白，看起来像翻到了空课表。
     */
    fun neighbors(week: Int, lastWeek: Int): List<Int> {
        if (week < 1) return emptyList()
        return listOf(week - 1, week + 1).filter { it >= 1 && (lastWeek < 1 || it <= lastWeek) }
    }

    /**
     * 邻页缓存应保留的周次：当前周 ±[window]，且**必须包含当前周**。
     * 翻一整个学期不该把每周 7 天都留在内存里；被裁掉的页落页时再读一次缓存。
     */
    fun retained(weeks: Collection<Int>, current: Int, window: Int = WINDOW): Set<Int> {
        val keep = weeks.filter { abs(it - current) <= window }.toMutableSet()
        if (current >= 1) keep += current
        return keep
    }
}
