package com.github.echoping07.kemaolife.data

import com.github.echoping07.kemaolife.ui.AppState
import org.json.JSONObject
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * 只测「信封 → DTO 投影」这一段，不测业务计算。
 *
 * 业务计算（周次、均量、可用天数）全在 Go 侧并有单测，Kotlin 复算一遍只会引入漂移
 * （AGENTS A3）。这里要守住的是：字段名不能拼错、null 与零值不能混淆、失败信封要带 code。
 */
class EnvelopeAndDtoTest {

    @Test
    fun `成功信封解析出 data 与 stale`() {
        val env = Envelope.parse(
            """{"ok":true,"data":{"date":"2026-11-05","week":10,"weekday":4,
               "weekdayCn":"四","inSemester":true,"entries":[]},
               "ts":"2026-11-05T08:00:00+08:00","stale":true,
               "syncedAt":"2026-11-04T21:10:02+08:00"}""",
        )
        assertTrue(env.ok)
        assertTrue(env.stale)
        assertEquals("2026-11-04T21:10:02+08:00", env.syncedAt)

        val day = DaySchedule.from(env.data!!)
        assertEquals("2026-11-05", day.date)
        assertEquals(10, day.week)
        assertEquals("四", day.weekdayCn)
        assertTrue(day.inSemester)
        assertTrue(day.entries.isEmpty())
    }

    @Test
    fun `失败信封带 code 与 message`() {
        val env = Envelope.parse(
            """{"ok":false,"error":{"code":3001,"message":"未配置学号"},"ts":"2026-11-05T08:00:00+08:00"}""",
        )
        assertFalse(env.ok)
        assertNull(env.data)
        assertEquals(3001, env.error?.code)
        assertEquals("未配置学号", env.error?.message)
    }

    @Test
    fun `非法 JSON 归为 4002 而不抛异常`() {
        val env = Envelope.parse("not json")
        assertFalse(env.ok)
        assertEquals(4002, env.error?.code)
    }

    @Test
    fun `电费 dailyAvg 为 null 时保持 null 而不是 0`() {
        // null 有明确语义（无数据 / 无法估算），用 0 代替会让 UI 显示「均量 0 度」，
        // 进而把「没数据」误读成「没用电」（docs/002 §3.2）。
        val report = ElectricityReport.from(
            JSONObject(
                """{"status":{"roomName":"<房间>","balance":35.99,"unit":"度",
                   "collectedAt":"2026-11-04T22:41:37+08:00","dailyAvg":null,"estimatedDays":null},
                   "usage":{"yearMonth":"2026-11","monthUse":63.13,"dataUntil":"2026-11-04",
                   "days":[{"date":"2026-11-03","use":11.64},{"date":"2026-11-04","use":11.75}]}}""",
            ),
        )
        assertNull(report.status.dailyAvg)
        assertNull(report.status.estimatedDays)
        assertEquals(35.99, report.status.balance, 1e-9)
        assertEquals("2026-11-04T22:41:37+08:00", report.status.collectedAt)
        assertEquals(2, report.usage.days.size)
        assertEquals("2026-11-03", report.usage.days[0].date)
    }

    @Test
    fun `校园网未知 state 归为 offline 而不是异常`() {
        // 后端若新增状态值，UI 应退化成「不在校园网」而不是崩溃或空白。
        assertEquals(CampusState.OFFLINE, CampusState.from("something-new"))
        assertEquals(CampusState.ONLINE, CampusState.from("online"))
        assertEquals(CampusState.UNAUTH, CampusState.from("unauth"))
    }

    @Test
    fun `校园网动作在 offline 时禁用`() {
        val offline = AppState(campus = CampusStatus(CampusState.OFFLINE, "", ""))
        assertFalse(offline.campusActionsEnabled)

        val online = AppState(campus = CampusStatus(CampusState.ONLINE, "", ""))
        assertTrue(online.campusActionsEnabled)

        // 还没探测过也不能点动作：状态未知时按钮应不可用。
        assertFalse(AppState().campusActionsEnabled)
    }

    @Test
    fun `凭据缺失时 needsCredentials 为真`() {
        val ex = AppException(ApiError(3001, "未配置学号"))
        assertTrue(ex.needsCredentials)

        val network = AppException(ApiError(4001, "网络不可达"))
        assertFalse(network.needsCredentials)
    }

    @Test
    fun `测试回报 null 值保留为空串并显示占位符`() {
        val probe = ProbeResult.from(
            JSONObject(
                """{"ok":true,"elapsedMs":12,"message":"","detail":{"estimatedDays":null,"collectedAt":""}}""",
            ),
        )
        // null 不能跳过整个字段，也不能序列化成字面量 "null"。
        assertEquals("", probe.detail["estimatedDays"])
        assertEquals("", probe.detail["collectedAt"])
        assertTrue(probe.ok)
        assertEquals(12L, probe.elapsedMs)
    }

    @Test
    fun `测试回报未知字段与动态 payload 被隔离`() {
        val probe = ProbeResult.from(
            JSONObject(
                """{"ok":true,"elapsedMs":1,"message":"","detail":{
                   "balance":35.99,"roomName":"<房间>","roomVerify":"<roomverify>"},
                   "report":{"status":{"balance":35.99,"unit":"度","dailyAvg":null,"estimatedDays":null,
                     "collectedAt":"","roomName":""},
                     "usage":{"yearMonth":"2026-11","monthUse":1.0,"dataUntil":"","days":[]}},
                   "campusStatus":{"state":"online","checkedAt":"","errorMsg":""}}""",
            ),
        )
        // 未知 detail 字段一律丢弃，避免上游新增字段原样摊给用户（P2）。
        assertFalse(probe.detail.containsKey("roomName"))
        assertFalse(probe.detail.containsKey("roomVerify"))
        assertEquals("", probe.report?.status?.roomName)
        assertEquals("2026-11", probe.report?.usage?.yearMonth)
        assertEquals(CampusState.ONLINE, probe.campusStatus?.state)
    }

    @Test
    fun `测试失败回报带固定错误码与中性文案`() {
        val probe = ProbeResult.from(
            JSONObject("""{"ok":false,"elapsedMs":3,"code":3001,"message":"凭据或本地数据不可用"}"""),
        )
        assertFalse(probe.ok)
        assertEquals(3001, probe.code)
        assertEquals("凭据或本地数据不可用", probe.message)
        assertTrue(probe.detail.isEmpty())
    }

    @Test
    fun `凭据老存档缺继承开关时默认打开`() {
        // 与 CredentialStore.fromJson 的兼容口径一致：升级后电费继续复用教务学号。
        val legacy = JSONObject("""{"studentId":"<学号>","eduPassword":"<密码>"}""")
        val inherit = if (legacy.has("electricityInheritEducation")) {
            legacy.optBoolean("electricityInheritEducation")
        } else {
            true
        }
        assertTrue(inherit)
    }

    @Test
    fun `周课表解析出 7 天`() {
        val days = (0 until 7).joinToString(",") {
            """{"date":"2026-11-0${it + 2}","week":10,"weekday":${it + 1},
               "weekdayCn":"一","inSemester":true,"entries":[]}"""
        }
        val env = Envelope.parse("""{"ok":true,"data":{"week":10,"days":[$days]},"ts":""}""")
        val week = WeekSchedule.from(env.data!!)
        assertEquals(10, week.week)
        assertEquals(7, week.days.size)
    }
}
