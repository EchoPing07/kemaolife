package com.github.echoping07.kemaolife.data

/**
 * 设置页里的全部凭据字段（docs/002 §5.3 tab3）。
 *
 * 分域与文档一致：
 *  - 教务：学号 + 密码 + 自动同步间隔
 *  - 电费：默认继承教务学号，可关闭后单独填学号；学校代码在此设置；无密码（api-docs/002 §1 无鉴权）
 *  - 校园网：独立账号 + 密码，不复用教务密码（D9 已定稿）
 */
data class Credentials(
    val studentId: String = "",
    val eduPassword: String = "",
    /** 电费继承教务学号（设置页默认打开）。 */
    val electricityInheritEducation: Boolean = true,
    /** 关闭继承后电费单独使用的学号；继承时恒为空。 */
    val electricityStudentId: String = "",
    val customerCode: String = DEFAULT_CUSTOMER_CODE,
    val campusUser: String = "",
    val campusPassword: String = "",
    val campusFingerprint: String = "",
) {
    /** 电费实际使用的学号，与 Go 側 `Config.ElectricityStudent()` 同一口径。 */
    val electricityAccount: String
        get() = if (electricityInheritEducation) studentId else electricityStudentId

    companion object {
        /** 学校代码内置默认值（学校事实，非个人隐私，docs/002 §1.1）。 */
        const val DEFAULT_CUSTOMER_CODE = "3126"
    }
}
