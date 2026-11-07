package com.github.echoping07.kemaolife.ui

import com.github.echoping07.kemaolife.data.Entry

/** 课表网格行数：教务为 12 小节（6 个大节，见 internal/core/timetable/protocol.go）。 */
internal const val TIMETABLE_ROWS = 12

/**
 * 一个课程块在网格里的位置。
 *
 * `lane` / `lanes` 只在同一天有撞课时用得上（横向平分列宽）；
 * 正常课表 99% 是单 lane。
 */
internal data class CourseBlock(
    val entry: Entry,
    val startRow: Int,
    val spanRows: Int,
    val lane: Int,
    val lanes: Int,
)

/**
 * 把某一天的课程投影成网格块。
 *
 * 这里**只做排版换算**，不做业务计算（AGENTS A3）：
 * `periods` 已由 Go 侧 `core` 归一化成 `1-2` / `5-8` 这种形状（docs/002 §3），
 * 本函数仅把首尾小节号映射成行区间与行数。
 */
internal fun layoutCourseBlocks(entries: List<Entry>): List<CourseBlock> {
    data class Item(val entry: Entry, val start: Int, val span: Int, var lane: Int = 0)

    val items = entries.mapNotNull { entry ->
        val nums = parsePeriodNumbers(entry.periods)
        if (nums.isEmpty()) return@mapNotNull null
        val start = (nums.min() - 1).coerceIn(0, TIMETABLE_ROWS - 1)
        val end = (nums.max() - 1).coerceIn(start, TIMETABLE_ROWS - 1)
        Item(entry, start, end - start + 1)
    }.sortedWith(compareBy({ it.start }, { it.span }))

    val out = ArrayList<CourseBlock>(items.size)
    var cluster = ArrayList<Item>()
    var clusterEnd = 0
    val laneEnds = ArrayList<Int>()

    fun flush() {
        if (cluster.isEmpty()) return
        val lanes = laneEnds.size.coerceAtLeast(1)
        cluster.forEach { item ->
            out += CourseBlock(item.entry, item.start, item.span, item.lane, lanes)
        }
        cluster = ArrayList()
        laneEnds.clear()
        clusterEnd = 0
    }

    for (item in items) {
        // 与当前重叠簇不再相交 → 收簇，后一块重新从 lane 0 开始。
        if (cluster.isNotEmpty() && item.start >= clusterEnd) flush()

        var lane = laneEnds.indexOfFirst { it <= item.start }
        if (lane < 0) {
            lane = laneEnds.size
            laneEnds += 0
        }
        laneEnds[lane] = item.start + item.span
        item.lane = lane
        cluster += item
        if (item.start + item.span > clusterEnd) clusterEnd = item.start + item.span
    }
    flush()
    return out
}

/**
 * `1-2` / `1,2` / `5-8` → 小节号列表；无法解析时返回空（该课不进网格）。
 *
 * 先把 `5-8` 展开成连续小节，再按逗号切分：教务的 periods 两种写法混用，
 * 只按逗号切会把 `5-8` 当成两个端点（块高就只有 1 行，画错）。
 */
internal fun parsePeriodNumbers(periods: String): List<Int> {
    val out = ArrayList<Int>()
    for (part in periods.split(',', '，', '、', ';')) {
        val trimmed = part.trim()
        if (trimmed.isEmpty()) continue
        val range = trimmed.split('-', '—', '~')
        if (range.size == 2) {
            val from = range[0].trim().toIntOrNull()
            val to = range[1].trim().toIntOrNull()
            if (from != null && to != null && from <= to) {
                for (n in from..to) if (n in 1..TIMETABLE_ROWS) out += n
                continue
            }
        }
        range.forEach { n -> n.trim().toIntOrNull()?.let { if (it in 1..TIMETABLE_ROWS) out += it } }
    }
    return out
}
